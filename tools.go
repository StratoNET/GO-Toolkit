package toolkit

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+-="

// Tools is used to instantiate this module. Any variable of this type will have access to all methods with the receiver *Tools
type Tools struct {
	AllowedFileTypes []string
	MaxFileSize      int
}

// RandomString returns string of random characters of length n, generated from randomStringSource
func (t *Tools) RandomString(n int) string {
	s, r := make([]rune, n), []rune(randomStringSource)
	for i := range s {
		p, err := rand.Prime(rand.Reader, len(r))
		if err != nil {
			log.Println(err)
		}
		x, y := p.Uint64(), uint64(len(r))
		s[i] = r[x%y]
	}

	return string(s)
}

// UploadedFile is used to hold information about an uploaded file
type UploadedFile struct {
	NewFileName      string
	OriginalFileName string
	FileSize         int64
}

// UploadFiles allows uploading multiple files in one action to a specified directory with, if required, random naming.
// A slice containing newly named files, original file names, file size is returned and potentially an error.
// If the optional last parameter is set to true then files are not renamed but retain their original file names.
func (t *Tools) UploadFiles(r *http.Request, uploadDir string, rename ...bool) ([]*UploadedFile, error) {
	// rename files by default or by whatever value in rename (which might exist)
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	var uploadedFiles []*UploadedFile

	// set default limit for MaxFileSize if not set by user (1GB)
	if t.MaxFileSize == 0 {
		t.MaxFileSize = 1024 * 1024 * 1024
	}

	// create new folder/directory if necessary
	err := t.CreateDirIfNotExist(uploadDir)
	if err != nil {
		return nil, err
	}

	err = r.ParseMultipartForm(int64(t.MaxFileSize))
	if err != nil {
		return nil, errors.New("uploaded file exceeds allowed maximum file size")
	}

	for _, fHeaders := range r.MultipartForm.File {
		for _, hdr := range fHeaders {
			uploadedFiles, err = func(uploadedFiles []*UploadedFile) ([]*UploadedFile, error) {
				// uploadedFile used to hold file extracted from request
				var uploadedFile UploadedFile
				inFile, err := hdr.Open()
				if err != nil {
					return nil, err
				}
				defer inFile.Close()

				// examine initial 512 bytes of inFile to determine type
				buff := make([]byte, 512)
				_, err = inFile.Read(buff)
				if err != nil {
					return nil, err
				}

				// check if file type is permitted, initiate isAllowed as false
				isAllowed := false
				// determine file type e.g. image/png, image/jpg etc.
				fileType := http.DetectContentType(buff)

				// check if AllowedFileTypes has been populated by user, else allow all file types !!!
				if len(t.AllowedFileTypes) > 0 {
					for _, ft := range t.AllowedFileTypes {
						if strings.EqualFold(fileType, ft) {
							isAllowed = true
						}
					}
				} else {
					isAllowed = true
				}

				if !isAllowed {
					return nil, errors.New("the uploaded file type is not permitted")
				}

				// now return to start of file having read first 512 bytes
				_, err = inFile.Seek(0, 0)
				if err != nil {
					return nil, err
				}

				// rename file or use original file name
				if renameFile {
					uploadedFile.NewFileName = fmt.Sprintf("%s%s", t.RandomString(32), filepath.Ext(hdr.Filename))
				} else {
					uploadedFile.NewFileName = hdr.Filename
				}
				uploadedFile.OriginalFileName = hdr.Filename

				// write file to disk in defined location (uploadDir)
				var outFile *os.File
				// defer file close at this point only if NOT on Windows
				if runtime.GOOS != "windows" {
					defer outFile.Close()
				}

				if outFile, err = os.Create(filepath.Join(uploadDir, uploadedFile.NewFileName)); err != nil {
					return nil, err
				} else {
					fileSize, err := io.Copy(outFile, inFile)
					if err != nil {
						return nil, err
					}
					uploadedFile.FileSize = fileSize
				}

				// explicitly close file to release its handle on Windows
				if runtime.GOOS == "windows" {
					outFile.Close()
				}

				uploadedFiles = append(uploadedFiles, &uploadedFile)

				return uploadedFiles, nil

			}(uploadedFiles)

			if err != nil {
				return uploadedFiles, err
			}

		}
	}

	return uploadedFiles, nil
}

// UploadOneFile convenience method which restricts to uploading only one file
func (t *Tools) UploadOneFile(r *http.Request, uploadDir string, rename ...bool) (*UploadedFile, error) {
	// rename file by default or by whatever value in rename (which might exist)
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	files, err := t.UploadFiles(r, uploadDir, renameFile)
	if err != nil {
		return nil, err
	}

	return files[0], nil
}

// CreateDirIfNotExist creates a directory and all necessary parents, if it does not exist
func (t *Tools) CreateDirIfNotExist(path string) error {
	const mode = 0755
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, mode)
		if err != nil {
			return err
		}
	}
	return nil
}

// Slugify creates a slug from a string
func (t *Tools) Slugify(s string) (string, error) {
	if s == "" {
		return "", errors.New("empty string not permitted")
	}

	var regex = regexp.MustCompile(`[^a-z\d]+`)
	slug := strings.Trim(regex.ReplaceAllString(strings.ToLower(s), "-"), "-")
	if len(slug) == 0 {
		return "", errors.New("after replacing characters, slug length is zero")
	}

	return slug, nil
}
