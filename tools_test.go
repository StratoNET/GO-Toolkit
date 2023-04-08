package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

// ============================== ALTERNATIVE HTTP CLIENT ===============================
// used to satisfy the requirement for an http client without having an actual remote API

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// NewTestClient provides 'pseudo' http client with no requirement for a remote API
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

// ======================================================================================

func TestTools_PushJSONToRemoteService(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		// set test request parameters
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("data from pretend remote API")),
			Header:     make(http.Header),
		}
	})

	var testTool Tools
	var retro struct {
		Rocker string `json:"rocker"`
	}
	retro.Rocker = "ok"

	_, _, err := testTool.PushJSONToRemoteService("http://stratosoft.uk/some/path", retro, client)
	if err != nil {
		t.Error("failed to call remote service:", err)
	}
}

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

var jsonTests = []struct {
	testName           string
	json               string
	maxSize            int
	allowUnknownFields bool
	errorExpected      bool
}{
	{testName: "valid JSON", json: `{"foo": "bar"}`, maxSize: 1024, allowUnknownFields: false, errorExpected: false},
	{testName: "badly formatted JSON", json: `{"foo":}`, maxSize: 1024, allowUnknownFields: false, errorExpected: true},
	{testName: "incorrect JSON type", json: `{"foo": 99}`, maxSize: 1024, allowUnknownFields: false, errorExpected: true},
	{testName: "multiple JSON files", json: `{"foo": "99"}{"bar": "66"}`, maxSize: 1024, allowUnknownFields: false, errorExpected: true},
	{testName: "empty request body", json: ``, maxSize: 1024, allowUnknownFields: false, errorExpected: true},
	{testName: "JSON syntax error", json: `{"foo": "bar}`, maxSize: 1024, allowUnknownFields: false, errorExpected: true},
	{testName: "unknown field in JSON", json: `{"foot": "bar"}`, maxSize: 1024, allowUnknownFields: false, errorExpected: true},
	{testName: "allow unknown field in JSON", json: `{"foot": "bar"}`, maxSize: 1024, allowUnknownFields: true, errorExpected: false},
	{testName: "missing field name in JSON", json: `{mars: "bar"}`, maxSize: 1024, allowUnknownFields: true, errorExpected: true},
	{testName: "JSON file too large", json: `{"foot": "bar"}`, maxSize: 4, allowUnknownFields: false, errorExpected: true},
	{testName: "not JSON", json: `oops, this is NOT JSON`, maxSize: 1024, allowUnknownFields: true, errorExpected: true},
}

func TestTools_ReadJSON(t *testing.T) {
	var testTool Tools

	for _, test := range jsonTests {
		// assign max file size
		testTool.MaxJSONPayloadSize = test.maxSize

		// allow unknown fields
		testTool.AllowUnknownFields = test.allowUnknownFields

		// declare variable to hold read JSON
		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		// create request using body
		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(test.json)))
		if err != nil {
			t.Log("Error: ", err)
		}

		// create recorder
		rr := httptest.NewRecorder()

		err = testTool.ReadJSON(rr, req, &decodedJSON)

		if test.errorExpected && err == nil {
			t.Errorf("%s: error expected but NOT generated", test.testName)
		}

		if !test.errorExpected && err != nil {
			t.Errorf("%s: error NOT expected but was generated: %s", test.testName, err.Error())
		}

		req.Body.Close()
	}
}

func TestTools_WriteJSON(t *testing.T) {
	var testTool Tools

	// create recorder
	rr := httptest.NewRecorder()

	payload := JSONResponse{
		Error:   false,
		Message: "from WriteJSON()",
	}

	headers := make(http.Header)
	headers.Add("ICE", "CREAM")

	err := testTool.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("failed to write JSON: %v", err)
	}
}

func TestTools_ErrorJSON(t *testing.T) {
	var testTool Tools

	// create recorder
	rr := httptest.NewRecorder()

	err := testTool.ErrorJSON(rr, errors.New("an error from ErrorJSON()"), http.StatusServiceUnavailable)
	if err != nil {
		t.Errorf("error JSON(): %v", err)
	}

	var payload JSONResponse

	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		t.Error("received error when decoding JSON:", err)
	}

	if !payload.Error {
		t.Error("error set to false in JSON... it should be true")
	}

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("incorrect status code returned: expected 503, received %d", rr.Code)
	}
}
