package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools

	rs := testTools.RandomString(10)
	if len(rs) != 10 {
		t.Error("incorrect length random string returned, should be 10 characters")
	}
}

var uploadTests = []struct {
	name             string
	allowedMIMETypes []string
	renameFile       bool
	errorExpected    bool
}{
	{name: "allowed MIME type, no rename", allowedMIMETypes: []string{"image/jpg", "image/jpeg", "image/png"}, renameFile: false, errorExpected: false},
	{name: "allowed MIME type, rename", allowedMIMETypes: []string{"image/jpg", "image/jpeg", "image/png"}, renameFile: true, errorExpected: false},
	{name: "not allowed MIME type, no rename", allowedMIMETypes: []string{"image/jpg", "image/jpeg"}, renameFile: false, errorExpected: true},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		// set up pipe to avoid buffering
		pr, pw := io.Pipe()
		// simulate multipart upload
		writer := multipart.NewWriter(pw)
		// ensure actions occur in sequence
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer writer.Close()
			defer wg.Done()

			/// create formdata field 'file'
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error("failed to create formdata file", err)
			}

			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error("failed to open formdata file", err)
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error("failed to write 'png' image", err)
			}
		}()

		// read from pipe which receives data
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = e.allowedMIMETypes

		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error("upload failed", err)
		}

		if !e.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: expected this file to exist: %s", e.name, err.Error())
			}

			// TODO: clean up (this doesn't actually work)
			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))
			// if err != nil {
			// 	t.Error("failed to delete uploaded test file", err)
			// }
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error expected but none received", e.name)
		}

		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	for _, e := range uploadTests {
		// set up pipe to avoid buffering
		pr, pw := io.Pipe()
		// simulate multipart upload
		writer := multipart.NewWriter(pw)

		go func() {
			defer writer.Close()

			/// create formdata field 'file'
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error("failed to create formdata file", err)
			}

			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error("failed to open formdata file", err)
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error("failed to write 'png' image", err)
			}
		}()

		// read from pipe which receives data
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools

		uploadedFiles, err := testTools.UploadOneFile(request, "./testdata/uploads/", e.renameFile)
		if err != nil {
			t.Error("upload failed", err)
		}

		if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName)); os.IsNotExist(err) {
			t.Errorf("expected this file to exist: %s", err.Error())
		}

		// clean up
		err = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName))
		if err != nil {
			t.Error("failed to delete uploaded test file", err)
		}
	}
}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var testTool Tools
	err := testTool.CreateDirIfNotExist("./testdata/newDir")
	if err != nil {
		t.Error("failed to create newDir", err)
	}

	// try creating new directory if it already exists
	err = testTool.CreateDirIfNotExist("./testdata/newDir")
	if err != nil {
		t.Error("failed trying to create newDir when it already exists", err)
	}

	// clean up
	err = os.Remove("./testdata/newDir")
	if err != nil {
		t.Error("failed to delete test directory newDir", err)
	}
}
