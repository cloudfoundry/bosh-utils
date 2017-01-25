package crypto_test

import (
	"bytes"
	"io"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/crypto"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	"os"
	"errors"
)

var _ = Describe("Algorithms", func() {

	Context("digest from a single reader", func() {
		var reader io.Reader

		BeforeEach(func() {
			reader = bytes.NewReader([]byte("something different"))
		})

		Context("sha1", func() {
			It("computes digest from a reader", func() {
				digest, err := DigestAlgorithmSHA1.CreateDigest(reader)
				Expect(err).ToNot(HaveOccurred())
				Expect(digest.String()).To(Equal("da7102c07515effc353226eac2be923c916c5c94"))
			})
		})

		Context("sha256", func() {
			It("computes digest from a reader", func() {
				digest, err := DigestAlgorithmSHA256.CreateDigest(reader)
				Expect(err).ToNot(HaveOccurred())
				Expect(digest.String()).To(Equal("sha256:73af606b33433fa3a699134b39d5f6bce1ab4a6d9ca3263d3300f31fc5776b12"))
			})
		})

		Context("sha512", func() {
			It("computes digest from a reader", func() {
				digest, err := DigestAlgorithmSHA512.CreateDigest(reader)
				Expect(err).ToNot(HaveOccurred())
				Expect(digest.String()).To(Equal("sha512:25b38e5cf4069979d4de934ed6cde40eceec1f7100fc2a5fc38d3569456ab2b7e191bbf5a78b533df94a77fcd48b8cb025a4b5db20720d1ac36ecd9af0c8989a"))
			})
		})

	})

	Context("digest from a set of files in a directory", func() {
		var fs *fakesys.FakeFileSystem

		BeforeEach(func() {
			fs = fakesys.NewFakeFileSystem()

			fs.RegisterOpenFile("/fake-templates-dir", &fakesys.FakeFile{
				Stats: &fakesys.FakeFileStats{FileType: fakesys.FakeFileTypeDir},
			})

			fs.RegisterOpenFile("/fake-templates-dir/file-1", &fakesys.FakeFile{
				Contents: []byte("fake-file-1-contents"),
			})

			fs.WriteFileString("/fake-templates-dir/file-1", "fake-file-1-contents")

			fs.RegisterOpenFile("/fake-templates-dir/config/file-2", &fakesys.FakeFile{
				Contents: []byte("fake-file-2-contents"),
			})
			fs.MkdirAll("/fake-templates-dir/config", os.ModePerm)
			fs.WriteFileString("/fake-templates-dir/config/file-2", "fake-file-2-contents")
		})

		Context("when walking sends an error", func() {
			BeforeEach(func() {
				fs.WalkErr = errors.New("You can't read that now")
			})

			It("returns an error", func() {
				_, err := DigestAlgorithmSHA1.CreateDigestFromDir("/fake-templates-dir", fs)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("You can't read that now"))
				Expect(err.Error()).To(ContainSubstring("Walking directory when calculating digest for "))
			})
		})

		Context("when opening a file returns an error", func() {
			BeforeEach(func() {
				fs.OpenFileErr = errors.New("fake-open-file-error")
			})

			It("returns an error", func() {
				_, err := DigestAlgorithmSHA1.CreateDigestFromDir("/fake-templates-dir", fs)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Opening file '/fake-templates-dir/config/file-2' for digest calculation: fake-open-file-error"))
			})
		})

		Context("sha1", func() {
			It("computes digest from a directory path", func() {
				digest, err := DigestAlgorithmSHA1.CreateDigestFromDir("/fake-templates-dir", fs)
				Expect(err).ToNot(HaveOccurred())
				Expect(digest.String()).To(Equal("bc0646cd41b98cd6c878db7a0573eca345f78200"))
			})
		})

		Context("sha256", func() {
			It("computes digest from a directory path", func() {
				digest, err := DigestAlgorithmSHA256.CreateDigestFromDir("/fake-templates-dir", fs)
				Expect(err).ToNot(HaveOccurred())
				Expect(digest.String()).To(Equal("sha256:bbe30bfd2c863f835829db9bc71c11e7194e7c61795ed99f325f677b6177ad84"))
			})
		})

		Context("sha512", func() {
			It("computes digest from a directory path", func() {
				digest, err := DigestAlgorithmSHA512.CreateDigestFromDir("/fake-templates-dir", fs)
				Expect(err).ToNot(HaveOccurred())
				Expect(digest.String()).To(Equal("sha512:6d82819b716b9295703bd4098ecf740816d6fd2624c66fa5f7b250e65643dd499cd4842e1c866872456e314ca44fe67688fdd07aa2b57e60d28fdd7b71cec2be"))
			})
		})
	})

})
