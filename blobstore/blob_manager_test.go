package blobstore_test

import (
	"io/ioutil"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"

	. "github.com/cloudfoundry/bosh-utils/blobstore"
)

var _ = Describe("Blob Manager", func() {
	var (
		fs       boshsys.FileSystem
		basePath string

		blobManager BlobManager
	)

	blobID := "105d33ae-655c-493d-bf9f-1df5cf3ca847"

	BeforeEach(func() {
		var err error
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fs = boshsys.NewOsFileSystem(logger)
		basePath, err = ioutil.TempDir("", "blobmanager")
		Expect(err).NotTo(HaveOccurred())

		blobManager = NewBlobManager(fs, basePath)
	})

	AfterEach(func() {
		os.RemoveAll(basePath)
	})

	It("can fetch what was written", func() {
		err := blobManager.Write(blobID, strings.NewReader("new data"))
		Expect(err).ToNot(HaveOccurred())

		readOnlyFile, err, status := blobManager.Fetch(blobID)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal(200))

		contents, err := fs.ReadFileString(readOnlyFile.Name())
		Expect(err).ToNot(HaveOccurred())
		Expect(contents).To(Equal("new data"))
	})

	It("can overwrite files", func() {
		err := blobManager.Write(blobID, strings.NewReader("old data"))
		Expect(err).ToNot(HaveOccurred())

		err = blobManager.Write(blobID, strings.NewReader("new data"))
		Expect(err).ToNot(HaveOccurred())

		readOnlyFile, err, status := blobManager.Fetch(blobID)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal(200))

		contents, err := fs.ReadFileString(readOnlyFile.Name())
		Expect(err).ToNot(HaveOccurred())
		Expect(contents).To(Equal("new data"))
	})

	It("can store multiple files", func() {
		err := blobManager.Write(blobID, strings.NewReader("data1"))
		Expect(err).ToNot(HaveOccurred())

		otherBlobId := "other-blob-id"
		err = blobManager.Write(otherBlobId, strings.NewReader("data2"))
		Expect(err).ToNot(HaveOccurred())

		readOnlyFile, err, status := blobManager.Fetch(blobID)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal(200))

		contents, err := fs.ReadFileString(readOnlyFile.Name())
		Expect(err).ToNot(HaveOccurred())
		Expect(contents).To(Equal("data1"))

		otherFile, err, status := blobManager.Fetch(otherBlobId)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(Equal(200))

		otherContents, err := fs.ReadFileString(otherFile.Name())
		Expect(err).ToNot(HaveOccurred())
		Expect(otherContents).To(Equal("data2"))
	})

	Describe("GetPath", func() {
		var sampleDigest boshcrypto.Digest

		BeforeEach(func() {
			correctCheckSum := "a17c9aaa61e80a1bf71d0d850af4e5baa9800bbd" // sha-1 of "data"
			sampleDigest = boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, correctCheckSum)
		})

		Context("when file requested exists in blobsPath", func() {
			BeforeEach(func() {
				err := blobManager.Write(blobID, strings.NewReader("data"))
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when file checksum matches provided checksum", func() {
				It("should return the path of a copy of the requested blob", func() {
					filename, err := blobManager.GetPath(blobID, sampleDigest)
					Expect(err).NotTo(HaveOccurred())

					contents, err := fs.ReadFileString(filename)
					Expect(err).NotTo(HaveOccurred())
					Expect(contents).To(Equal("data"))
				})
			})

			Context("when file checksum does NOT match provided checksum", func() {
				It("should return an error", func() {
					bogusDigest := boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "bogus-sha")
					_, err := blobManager.GetPath(blobID, bogusDigest)
					Expect(err).To(MatchError(ContainSubstring(blobID)))
					Expect(err).To(MatchError(ContainSubstring(sampleDigest.String())))
					Expect(err).To(MatchError(ContainSubstring(bogusDigest.String())))
				})
			})

			It("does not allow modifications made to the returned path to affect the original file", func() {
				path, err := blobManager.GetPath(blobID, sampleDigest)
				Expect(err).NotTo(HaveOccurred())

				err = fs.WriteFileString(path, "overwriting!")
				Expect(err).NotTo(HaveOccurred())

				readOnlyFile, err, status := blobManager.Fetch(blobID)
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal(200))

				contents, err := fs.ReadFileString(readOnlyFile.Name())
				Expect(err).ToNot(HaveOccurred())
				Expect(contents).To(Equal("data"))
			})

			It("puts the temporary file inside the work directory to make sure that no files leak out if it's mounted on a tmpfs", func() {
				path, err := blobManager.GetPath(blobID, sampleDigest)
				Expect(err).NotTo(HaveOccurred())

				Expect(path).To(HavePrefix(basePath))
			})
		})

		Context("when file requested does not exist in blobsPath", func() {
			It("returns an error", func() {
				_, err := blobManager.GetPath("missing", sampleDigest)
				Expect(err).To(MatchError("Blob 'missing' not found"))
			})
		})
	})

	Describe("Delete", func() {
		Context("when file to be deleted exists in blobsPath", func() {
			BeforeEach(func() {
				err := blobManager.Write(blobID, strings.NewReader("data"))
				Expect(err).ToNot(HaveOccurred())
			})

			It("should delete the blob", func() {
				exists := blobManager.BlobExists(blobID)
				Expect(exists).To(BeTrue())

				err := blobManager.Delete(blobID)
				Expect(err).NotTo(HaveOccurred())

				exists = blobManager.BlobExists(blobID)
				Expect(exists).To(BeFalse())

				_, err, status := blobManager.Fetch(blobID)
				Expect(err).To(HaveOccurred())
				Expect(status).To(Equal(404))
			})
		})

		Context("when file to be deleted does not exist in blobsPath", func() {
			It("does not error", func() {
				err := blobManager.Delete("hello-i-am-no-one")
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("BlobExists", func() {
		Context("when blob requested exists in blobsPath", func() {
			BeforeEach(func() {
				err := blobManager.Write(blobID, strings.NewReader("super-smurf-content"))
				Expect(err).To(BeNil())
			})

			It("returns true", func() {
				exists := blobManager.BlobExists(blobID)
				Expect(exists).To(BeTrue())
			})
		})

		Context("when blob requested does NOT exist in blobsPath", func() {
			It("returns false", func() {
				exists := blobManager.BlobExists("blob-id-does-not-exist")
				Expect(exists).To(BeFalse())
			})
		})
	})
})
