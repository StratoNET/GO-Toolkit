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

var slugTests = []struct {
	testName      string
	toSlug        string
	expectedSlug  string
	errorExpected bool
}{
	{testName: "valid slug string", toSlug: "mary had a little lamb", expectedSlug: "mary-had-a-little-lamb", errorExpected: false},
	{testName: "empty slug string", toSlug: "", expectedSlug: "", errorExpected: true},
	{testName: "messy slug string", toSlug: "$All I want for Christmas is YOU & a Ferrari and an RTX 4090 !%", expectedSlug: "all-i-want-for-christmas-is-you-a-ferrari-and-an-rtx-4090", errorExpected: false},
	{testName: "greek slug string", toSlug: "Γειά σου Κόσμε", expectedSlug: "", errorExpected: true},
	{testName: "greek & roman characters slug string", toSlug: "Hello Γειά σου Κόσμε World", expectedSlug: "hello-world", errorExpected: false},
}

func TestTools_Slugify(t *testing.T) {
	var testTool Tools

	for _, e := range slugTests {
		slug, err := testTool.Slugify(e.toSlug)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: error occurred when none was expected: %s", e.testName, err.Error())
		}

		if !e.errorExpected && slug != e.expectedSlug {
			t.Errorf("%s: incorrect slug generated; expected %s but returned %s", e.testName, e.expectedSlug, slug)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	wr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	var testTool Tools

	testTool.DownloadStaticFile(wr, req, "./testdata", "image.jpg", "ralf.jpg")

	result := wr.Result()
	defer result.Body.Close()

	// check file length
	if result.Header["Content-Length"][0] != "55715" {
		t.Error("incorrect content length:", result.Header["Content-Length"][0])
	}

	// check content disposition
	if result.Header["Content-Disposition"][0] != "attachment; filename=\"ralf.jpg\"" {
		t.Error("incorrect content disposition:", result.Header["Content-Disposition"][0])
	}

	_, err := io.ReadAll(result.Body)
	if err != nil {
		t.Error(err)
	}
}
