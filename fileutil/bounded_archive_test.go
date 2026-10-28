package fileutil_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

type archiveMember struct {
	header *tar.Header
	body   string
}

func regularMember(name, body string) archiveMember {
	return archiveMember{header: &tar.Header{Name: name, Mode: 0600, Size: int64(len(body)), Typeflag: tar.TypeReg}, body: body}
}

func tarBytes(members ...archiveMember) []byte {
	var raw bytes.Buffer
	writer := tar.NewWriter(&raw)
	for _, member := range members {
		Expect(writer.WriteHeader(member.header)).To(Succeed())
		if member.header.Size > 0 {
			_, err := writer.Write([]byte(member.body))
			Expect(err).ToNot(HaveOccurred())
		}
	}
	Expect(writer.Close()).To(Succeed())
	return raw.Bytes()
}

func gzipBytes(data []byte) []byte {
	var zipped bytes.Buffer
	writer := gzip.NewWriter(&zipped)
	_, err := writer.Write(data)
	Expect(err).ToNot(HaveOccurred())
	Expect(writer.Close()).To(Succeed())
	return zipped.Bytes()
}

func writeArchive(data []byte) string {
	path := filepath.Join(GinkgoT().TempDir(), "archive.tgz")
	Expect(os.WriteFile(path, data, 0600)).To(Succeed())
	return path
}

func gzipArchive(members ...archiveMember) string {
	return writeArchive(gzipBytes(tarBytes(members...)))
}

// oldGNUSparseArchive builds an old GNU format sparse member (type 'S') holding
// "data" at offset 4096 of an 8192 byte file. archive/tar can read but not write it.
func oldGNUSparseArchive() string {
	block := make([]byte, 512)
	put := func(offset int, value string) { copy(block[offset:], value) }
	octal := func(offset, width int, value int64) { put(offset, fmt.Sprintf("%0*o", width-1, value)) }
	put(0, "sparse")
	octal(100, 8, 0644)
	octal(108, 8, 0)
	octal(116, 8, 0)
	octal(124, 12, 4)
	octal(136, 12, 0)
	block[156] = tar.TypeGNUSparse
	put(257, "ustar  \x00")
	octal(386, 12, 4096)
	octal(398, 12, 4)
	octal(483, 12, 8192)
	put(148, "        ")
	sum := 0
	for _, b := range block {
		sum += int(b)
	}
	put(148, fmt.Sprintf("%06o\x00 ", sum))

	data := make([]byte, 512)
	copy(data, "data")
	archive := append(append(block, data...), make([]byte, 1024)...)
	return writeArchive(gzipBytes(archive))
}

func entriesIn(dir string) []os.DirEntry {
	entries, err := os.ReadDir(dir)
	Expect(err).ToNot(HaveOccurred())
	return entries
}

var _ = Describe("tarballCompressor DecompressFileToDir with archive limits", func() {
	var (
		compressor  Compressor
		destination string
		limits      CompressorOptions
	)

	BeforeEach(func() {
		fs := boshsys.NewOsFileSystem(boshlog.NewLogger(boshlog.LevelNone))
		compressor = NewTarballCompressor(nil, fs)
		destination = GinkgoT().TempDir()
		limits = CompressorOptions{MaxExpandedBytes: 100, MaxArchiveMembers: 10}
	})

	It("uses the tar command when no limits are given and not when they are", func() {
		cmdRunner := fakesys.NewFakeCmdRunner()
		fs := boshsys.NewOsFileSystem(boshlog.NewLogger(boshlog.LevelNone))
		archive := gzipArchive(regularMember("file", "content"))

		Expect(NewTarballCompressor(cmdRunner, fs).DecompressFileToDir(archive, destination, CompressorOptions{})).To(Succeed())
		Expect(cmdRunner.RunCommands).To(HaveLen(1))
		Expect(cmdRunner.RunCommands[0][0]).To(Equal("tar"))

		Expect(NewTarballCompressor(cmdRunner, fs).DecompressFileToDir(archive, destination, limits)).To(Succeed())
		Expect(cmdRunner.RunCommands).To(HaveLen(1))
		Expect(os.ReadFile(filepath.Join(destination, "file"))).To(Equal([]byte("content")))
	})

	It("rejects a member over the byte limit before creating it", func() {
		member := regularMember("oversized", strings.Repeat("x", 17))
		limits.MaxExpandedBytes = 16
		for _, archive := range []string{writeArchive(tarBytes(member)), gzipArchive(member)} {
			err := compressor.DecompressFileToDir(archive, destination, limits)
			Expect(err).To(MatchError(ContainSubstring("exceeds 16 expanded bytes")))
			Expect(entriesIn(destination)).To(BeEmpty())
		}
	})

	It("stops before the member that exceeds the member limit", func() {
		archive := gzipArchive(regularMember("one", "1"), regularMember("two", "2"), regularMember("three", "3"))
		limits.MaxArchiveMembers = 2

		err := compressor.DecompressFileToDir(archive, destination, limits)
		Expect(err).To(MatchError(ContainSubstring("exceeds 2 members")))
		Expect(os.ReadFile(filepath.Join(destination, "two"))).To(Equal([]byte("2")))
		Expect(filepath.Join(destination, "three")).ToNot(BeAnExistingFile())
	})

	It("bounds tar padding and trailing data that carry no file payload", func() {
		raw := append(tarBytes(regularMember("file", "x")), make([]byte, 2*1024*1024)...)
		limits.MaxArchiveMembers = 1

		err := compressor.DecompressFileToDir(writeArchive(gzipBytes(raw)), destination, limits)
		Expect(err).To(MatchError(ContainSubstring("byte/metadata budget")))
	})

	It("rejects unsafe member names and links", func() {
		unsafe := []archiveMember{
			regularMember("../outside", "x"),
			regularMember("dir/../../outside", "x"),
			regularMember("/absolute", "x"),
			regularMember(`back\slash`, "x"),
			regularMember("c:colon", "x"),
			{header: &tar.Header{Name: "link", Linkname: "../outside", Typeflag: tar.TypeSymlink}},
			{header: &tar.Header{Name: "dir/link", Linkname: "../../outside", Typeflag: tar.TypeSymlink}},
			{header: &tar.Header{Name: "link", Linkname: "/etc/passwd", Typeflag: tar.TypeSymlink}},
			{header: &tar.Header{Name: "hardlink", Linkname: "../outside", Typeflag: tar.TypeLink}},
			{header: &tar.Header{Name: "fifo", Mode: 0600, Typeflag: tar.TypeFifo}},
			{header: &tar.Header{Name: "device", Mode: 0600, Typeflag: tar.TypeChar}},
		}
		for _, member := range unsafe {
			dir := GinkgoT().TempDir()
			err := compressor.DecompressFileToDir(gzipArchive(member), dir, limits)
			Expect(err).To(HaveOccurred(), member.header.Name)
			_, err = os.Lstat(filepath.Join(dir, filepath.FromSlash(member.header.Name)))
			Expect(err).To(HaveOccurred(), member.header.Name)
			Expect(filepath.Join(filepath.Dir(dir), "outside")).ToNot(BeAnExistingFile())
		}
	})

	It("rejects options it cannot honour", func() {
		archive := gzipArchive(regularMember("file", "x"))
		for _, options := range []CompressorOptions{
			{MaxExpandedBytes: 100},
			{MaxArchiveMembers: 10},
			{MaxExpandedBytes: 100, MaxArchiveMembers: 10, SameOwner: true},
			{MaxExpandedBytes: 100, MaxArchiveMembers: 10, StripComponents: 1},
			{MaxExpandedBytes: 100, MaxArchiveMembers: 10, PathInArchive: "file"},
		} {
			Expect(compressor.DecompressFileToDir(archive, destination, options)).ToNot(Succeed())
		}
		Expect(entriesIn(destination)).To(BeEmpty())
	})

	It("extracts directories, legacy regular files, long names and sparse files", func() {
		gnuName := strings.Repeat("d", 20) + "/" + strings.Repeat("g", 110)
		paxName := strings.Repeat("d", 20) + "/" + strings.Repeat("p", 110)
		legacy := tarBytes(regularMember("legacy", "old"))
		legacy[156] = 0 // pre-POSIX regular file type flag
		legacy[148] = 0
		copy(legacy[148:156], "        ")
		sum := 0
		for _, b := range legacy[:512] {
			sum += int(b)
		}
		copy(legacy[148:156], fmt.Sprintf("%06o\x00 ", sum))

		archive := gzipArchive(
			archiveMember{header: &tar.Header{Name: "dir/", Mode: 0755, Typeflag: tar.TypeDir}},
			regularMember("dir/file", "content"),
			archiveMember{header: &tar.Header{Name: gnuName, Mode: 0600, Size: 3, Typeflag: tar.TypeReg, Format: tar.FormatGNU}, body: "gnu"},
			archiveMember{header: &tar.Header{Name: paxName, Mode: 0600, Size: 3, Typeflag: tar.TypeReg, Format: tar.FormatPAX}, body: "pax"},
		)
		Expect(compressor.DecompressFileToDir(archive, destination, limits)).To(Succeed())
		Expect(os.ReadFile(filepath.Join(destination, "dir", "file"))).To(Equal([]byte("content")))
		Expect(os.ReadFile(filepath.Join(destination, filepath.FromSlash(gnuName)))).To(Equal([]byte("gnu")))
		Expect(os.ReadFile(filepath.Join(destination, filepath.FromSlash(paxName)))).To(Equal([]byte("pax")))

		Expect(compressor.DecompressFileToDir(writeArchive(gzipBytes(legacy)), destination, limits)).To(Succeed())
		Expect(os.ReadFile(filepath.Join(destination, "legacy"))).To(Equal([]byte("old")))

		limits.MaxExpandedBytes = 10000
		Expect(compressor.DecompressFileToDir(oldGNUSparseArchive(), destination, limits)).To(Succeed())
		sparse, err := os.ReadFile(filepath.Join(destination, "sparse"))
		Expect(err).ToNot(HaveOccurred())
		Expect(sparse).To(HaveLen(8192))
		Expect(string(sparse[4096:4100])).To(Equal("data"))
		Expect(bytes.Count(sparse, []byte{0})).To(Equal(8192 - 4))
	})

	It("counts the logical size of sparse files against the byte limit", func() {
		limits.MaxExpandedBytes = 8191
		err := compressor.DecompressFileToDir(oldGNUSparseArchive(), destination, limits)
		Expect(err).To(MatchError(ContainSubstring("expanded bytes")))
	})

	Context("on file systems with POSIX permissions and links", func() {
		BeforeEach(func() {
			if runtime.GOOS == "windows" {
				Skip("symlinks and permission bits are not portable to Windows")
			}
		})

		It("extracts permissions, internal symlinks and hardlinks", func() {
			archive := gzipArchive(
				archiveMember{header: &tar.Header{Name: "bin/", Mode: 0750, Typeflag: tar.TypeDir}},
				archiveMember{header: &tar.Header{Name: "bin/run", Mode: 0755, Size: 2, Typeflag: tar.TypeReg}, body: "#!"},
				archiveMember{header: &tar.Header{Name: "secret", Mode: 0640, Size: 1, Typeflag: tar.TypeReg}, body: "s"},
				archiveMember{header: &tar.Header{Name: "bin/link", Linkname: "../secret", Typeflag: tar.TypeSymlink}},
				archiveMember{header: &tar.Header{Name: "hard", Linkname: "bin/run", Typeflag: tar.TypeLink}},
			)
			Expect(compressor.DecompressFileToDir(archive, destination, limits)).To(Succeed())

			info, err := os.Stat(filepath.Join(destination, "bin", "run"))
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Mode().Perm() & 0700).To(Equal(os.FileMode(0700)))
			info, err = os.Stat(filepath.Join(destination, "bin"))
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Mode().Perm() & 0750).To(Equal(os.FileMode(0750)))

			target, err := os.Readlink(filepath.Join(destination, "bin", "link"))
			Expect(err).ToNot(HaveOccurred())
			Expect(target).To(Equal("../secret"))

			hard, err := os.Stat(filepath.Join(destination, "hard"))
			Expect(err).ToNot(HaveOccurred())
			run, err := os.Stat(filepath.Join(destination, "bin", "run"))
			Expect(err).ToNot(HaveOccurred())
			Expect(os.SameFile(hard, run)).To(BeTrue())
		})

		It("checks manifest size before writing and through link aliases", func() {
			limits.MaxManifestBytes = 3
			archives := map[string]string{
				"regular":  gzipArchive(regularMember("release.MF", "data")),
				"symlink":  gzipArchive(regularMember("data", "data"), archiveMember{header: &tar.Header{Name: "release.MF", Linkname: "data", Typeflag: tar.TypeSymlink}}),
				"hardlink": gzipArchive(regularMember("data", "data"), archiveMember{header: &tar.Header{Name: "job.MF", Linkname: "data", Typeflag: tar.TypeLink}}),
			}
			for kind, archive := range archives {
				dir := GinkgoT().TempDir()
				err := compressor.DecompressFileToDir(archive, dir, limits)
				Expect(err).To(MatchError(ContainSubstring("metadata byte limit")), kind)
				if kind == "regular" {
					Expect(entriesIn(dir)).To(BeEmpty())
				}
			}
		})
	})

	Context("with a shared archive budget", func() {
		It("applies the expanded byte budget across extractions", func() {
			budget, err := NewArchiveBudget(9, 7)
			Expect(err).ToNot(HaveOccurred())
			limits.ArchiveBudget = budget

			Expect(compressor.DecompressFileToDir(gzipArchive(regularMember("first", strings.Repeat("x", 8))), destination, limits)).To(Succeed())

			second := GinkgoT().TempDir()
			err = compressor.DecompressFileToDir(gzipArchive(regularMember("second", "xx")), second, limits)
			Expect(err).To(MatchError(ContainSubstring("combined archives exceed expanded bytes budget")))
			Expect(entriesIn(second)).To(BeEmpty())
		})

		It("applies the member budget across extractions", func() {
			budget, err := NewArchiveBudget(100, 2)
			Expect(err).ToNot(HaveOccurred())
			limits.ArchiveBudget = budget

			Expect(compressor.DecompressFileToDir(gzipArchive(regularMember("one", "1"), regularMember("two", "2")), destination, limits)).To(Succeed())
			err = compressor.DecompressFileToDir(gzipArchive(regularMember("three", "3")), GinkgoT().TempDir(), limits)
			Expect(err).To(MatchError(ContainSubstring("combined archives exceed member budget")))
		})

		It("applies the manifest budget across extractions", func() {
			budget, err := NewArchiveBudget(100, 10, 3)
			Expect(err).ToNot(HaveOccurred())
			limits.ArchiveBudget = budget
			limits.MaxManifestBytes = 3

			Expect(compressor.DecompressFileToDir(gzipArchive(regularMember("job.MF", "{}")), destination, limits)).To(Succeed())
			err = compressor.DecompressFileToDir(gzipArchive(regularMember("job.MF", "{}")), GinkgoT().TempDir(), limits)
			Expect(err).To(MatchError(ContainSubstring("combined release manifests exceed metadata byte budget")))
		})

		It("rejects invalid budgets", func() {
			for _, limit := range [][2]int64{{0, 1}, {1, 0}, {-1, 1}, {1 << 62, 1 << 53}} {
				_, err := NewArchiveBudget(limit[0], int(limit[1]))
				Expect(err).To(HaveOccurred(), fmt.Sprint(limit))
			}
		})
	})

	It("extracts archives compressed with external tools", func() {
		raw := writeArchive(tarBytes(regularMember("data", "data")))
		for _, program := range []string{"bzip2", "xz", "zstd"} {
			if _, err := exec.LookPath(program); err != nil {
				GinkgoWriter.Printf("skipping %s: not installed\n", program)
				continue
			}
			compressed, err := exec.Command(program, "-c", raw).Output()
			Expect(err).ToNot(HaveOccurred())
			dir := GinkgoT().TempDir()
			Expect(compressor.DecompressFileToDir(writeArchive(compressed), dir, limits)).To(Succeed(), program)
			Expect(os.ReadFile(filepath.Join(dir, "data"))).To(Equal([]byte("data")), program)
		}
	})
})
