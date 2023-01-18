package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image/jpeg"
	"io"
	"io/fs"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/syrinsecurity/gologger"
	"golang.org/x/image/tiff"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	httpTimeoutDuration = 30 * time.Minute
)

type PeroProc struct {
	id         string
	input      string
	output     string
	numOfFiles int
	doneFiles  int
}

func changeNumOfDoneFiles(id string, num int) {
	runningProcesses[id].doneFiles = num
	db.Model(&Logs{}).Where("log_id = ?", id).Update("state", strconv.Itoa(runningProcesses[id].doneFiles)+"/"+strconv.Itoa(runningProcesses[id].numOfFiles))
}

var (
	imgExts  = [...]string{".tiff", ".tif", ".jpg", ".png", ".jp2", ".jp2k"}
	engineID int
	apiKey   string
	endpoint string

	runningProcesses        = make(map[string]*PeroProc)
	processesForTermination = make(map[string]bool)
)

func RunPero(input string, output string, engineId int, pull_only bool) {

	engineID = engineId

	apiKey = os.Getenv("API_KEY")
	endpoint = os.Getenv("ENDPOINT")

	dir := input
	if dir == "" {
		fmt.Println("-d switch is mandatory  <br>")
		flag.Usage()
		return
	}

	if !isDir(dir) {
		fmt.Println("file is not a directory or does not exist  <br>")
		return
	}

	// run through all available image files
	var files []string
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && isImage(path) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		fmt.Println("error while examining directory, check permissions and contents of this directory  <br>")
		return
	}

	// run ocr
	requestId := postPreprocessRequest(files)

	runningProcesses[requestId] = &PeroProc{
		id:         requestId,
		input:      input,
		output:     output,
		numOfFiles: len(files),
	}

	logger, err := gologger.New("./templates/logs/"+requestId+".log", 200)
	if err != nil {
		fmt.Printf("Panic, %e\n", err)
	}

	if requestId == "" {
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " ", os.Stderr, "Error for post_preprocess_request unknown response  <br>")
		return
	}

	process := &Logs{
		LogID:     requestId,
		Status:    "Uploading",
		Type:      "Pero",
		Input:     input,
		ShowBtn:   true,
		TimeStart: "Not started yet",
		TimeEnd:   "Not ended yet",
		EngineID:  engineId,
	}

	createdProcess := db.Create(&process)
	err = createdProcess.Error
	if err != nil {
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " ", err, "<br>")
	}

	logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " OCR for ", dir, " RequestId ", requestId, "<br>\n")

	time.Sleep(1 * time.Second)
	logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " starting file upload... <br>")
	db.Model(&Logs{}).Where("log_id = ?", requestId).Update("time_start", time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"))
	uploadedFiles := uploadFilesForOcr(files, requestId)
	logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " waiting for ocr to be done... <br>")
	changeNumOfDoneFiles(requestId, 0)
	db.Model(&Logs{}).Where("log_id = ?", requestId).Update("status", "Running")
	waitForOcrFinish(uploadedFiles, requestId)

	if processesForTermination[requestId] {
		delete(processesForTermination, requestId)
		db.Model(&Logs{}).Where("log_id = ?", requestId).Update("status", "Terminated")
		db.Model(&Logs{}).Where("log_id = ?", requestId).Update("time_end", time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"))
		db.Model(&Logs{}).Where("log_id = ?", requestId).Update("show_btn", false)
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Process", requestId, " has been terminated <br>\n")
		return
	}

	logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " downloading files from ocr... <br>")
	downloadOcrAlto(uploadedFiles, requestId)
	logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " batch complete <br>")
	runningProcesses[requestId] = nil
	delete(runningProcesses, requestId)
	db.Model(&Logs{}).Where("log_id = ?", requestId).Update("status", "Finished")
	db.Model(&Logs{}).Where("log_id = ?", requestId).Update("time_end", time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"))
	db.Model(&Logs{}).Where("log_id = ?", requestId).Update("show_btn", false)
}

func CancelOcrRequest(id string) {
	logger, err := gologger.New("./templates/logs/"+id+".log", 200)
	if err != nil {
		fmt.Printf("Panic, %e\n", err)
	}
	processesForTermination[id] = true
	req, _ := http.NewRequest("POST", endpoint+"cancel_request/"+id, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("api-key", apiKey)
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " ", os.Stderr, "Error creating request for CancelOcrRequest <br>")
		return
	}
	if res.StatusCode == http.StatusOK {
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " OK cancel req: "+id+" <br>")
		return
	}
	logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Error cancel request: ", id, " server returned status ", res.StatusCode, " <br>\n")
}

func downloadOcrAlto(files []string, requestId string) {
	logger, err := gologger.New("./templates/logs/"+requestId+".log", 200)
	if err != nil {
		fmt.Printf("Panic, %e\n", err)
	}
	for _, file := range files {
		req, _ := http.NewRequest("GET", endpoint+"download_results/"+requestId+"/"+ocrFilename(file)+"/txt", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Add("api-key", apiKey)
		client := http.Client{}
		res, err := client.Do(req)
		if err != nil {
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " ", os.Stderr, "Error creating request for request_status <br>")
			return
		}
		if res.StatusCode == http.StatusOK {
			filename := strings.TrimSuffix(file, filepath.Ext(file)) + ".txt"
			txt, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0775)
			if err != nil {
				logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " ", os.Stderr, "error creating file: "+filename+"<br>")
				continue
			}
			b, _ := ioutil.ReadAll(res.Body)
			txt.Write(b)
			txt.Close()
		}
		req, _ = http.NewRequest("GET", endpoint+"download_results/"+requestId+"/"+ocrFilename(file)+"/alto", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Add("api-key", apiKey)
		client = http.Client{}
		res, err = client.Do(req)
		if err != nil {
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " ", os.Stderr, "Error creating request for request_status <br>")
			return
		}
		if res.StatusCode == http.StatusOK {
			filename := strings.TrimSuffix(file, filepath.Ext(file)) + ".xml"
			txt, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0775)
			if err != nil {
				logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " ", os.Stderr, "error creating file: "+filename+"<br>")
				continue
			}
			b, _ := ioutil.ReadAll(res.Body)
			txt.Write(b)
			txt.Close()
		}
	}
}

func isDir(pathFile string) bool {
	if pathAbs, err := filepath.Abs(pathFile); err != nil {
		return false
	} else if fileInfo, err := os.Stat(pathAbs); os.IsNotExist(err) || !fileInfo.IsDir() {
		return false
	}

	return true
}

// isImage checks if file extension is on the list
//
// If supported image extension is found returns true
func isImage(file string) bool {
	ext := filepath.Ext(file)
	for _, imgExt := range imgExts {
		if imgExt == ext {
			return true
		}
	}
	return false
}

// postPreprocessRequest prepares for upload images to ocr
//
// returns request id
func postPreprocessRequest(files []string) string {
	var data = make(map[string]interface{})
	data["engine"] = engineID
	var imgs = make(map[string]interface{})
	for _, file := range files {
		imgs[ocrFilename(file)] = nil
	}
	data["images"] = imgs
	j, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshaling data for post_preprocess_request <br>")
		return ""
	}
	req, _ := http.NewRequest("POST", endpoint+"post_processing_request", bytes.NewReader(j))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("api-key", apiKey)
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Error creating request for post_preprocess_request <br>")
		return ""
	}
	if res.StatusCode != http.StatusOK {
		switch res.StatusCode {
		case 404:
			fmt.Println("Error for post_preprocess_request server responded 404 - ocr engine not found <br>")
			return ""
		case 422:
			fmt.Println("Error for post_preprocess_request server responded 422 - bad json data <br>")
			return ""
		default:
			fmt.Println("Error for post_preprocess_request server responded " + strconv.Itoa(res.StatusCode) + " (this error is not implemented) <br>")
			return ""
		}
	}

	var respData map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&respData)
	if err != nil {
		fmt.Println("Error for post_preprocess_request server not responded using json, cannot continue <br>")
		return ""
	}

	if respData["status"] == "success" {
		return respData["request_id"].(string)
	}

	return ""
}

// uploadFilesForOcr is self explanatory :P
//
// RequestID returned by PERO server is needed
func uploadFilesForOcr(files []string, requestId string) []string {
	logger, err := gologger.New("./templates/logs/"+requestId+".log", 200)
	if err != nil {
		fmt.Printf("Panic, %e\n", err)
	}
	c := http.Client{}
	var uploadedFiles []string
	for _, file := range files {
		if !tiffExtensionTest(filepath.Ext(file)) {
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), "Skipping file: ", file, ". Not a tiff file.<br>\n")
			continue
		}
		if processesForTermination[requestId] {
			break
		}
		f := convertTiffToJpg(file, requestId)
		if f == "" {
			f = file
		}
		b, w := createMultipartFormData("file", f)
		req, _ := http.NewRequest("POST", endpoint+"upload_image/"+requestId+"/"+ocrFilename(file), &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Add("api-key", apiKey)
		res, err := c.Do(req)
		if file != f {
			// remove temporary file
			os.Remove(f)
		}
		if err != nil {
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Error sending file: ", file, " to PERO <br>\n")
			changeNumOfDoneFiles(requestId, runningProcesses[requestId].doneFiles+1)
			continue
		}
		if res.StatusCode != http.StatusOK {
			var respData map[string]interface{}
			json.NewDecoder(res.Body).Decode(&respData)
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Error server status code ", res.StatusCode, " with message: ", respData["message"], " <br>\n")
		} else {
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " OK upload: ", file, " <br>\n")
			uploadedFiles = append(uploadedFiles, file)
		}
		changeNumOfDoneFiles(requestId, runningProcesses[requestId].doneFiles+1)
		w.Close()
		res.Body.Close()
		runtime.GC()
	}
	return uploadedFiles
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}

func normalizeString(s string) string {
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	result, _, _ := transform.String(t, s)
	return result
}

func ocrFilename(file string) string {
	return normalizeString(strings.ReplaceAll(filepath.Base(file), " ", ""))
}

func createMultipartFormData(fieldName, fileName string) (bytes.Buffer, *multipart.Writer) {
	var b bytes.Buffer
	var err error
	w := multipart.NewWriter(&b)
	var fw io.Writer
	file := mustOpen(fileName)
	if file == nil {
		w.Close()
		file.Close()
		return b, w
	}
	if fw, err = w.CreateFormFile(fieldName, file.Name()); err != nil {
		fmt.Printf("Error creating writer: %v <br>\n", err)
	}
	if _, err = io.Copy(fw, file); err != nil {
		fmt.Printf("Error with io.Copy: %v <br>\n", err)
	}
	w.Close()
	file.Close()
	return b, w
}

func waitForOcrFinish(files []string, requestId string) {
	logger, err := gologger.New("./templates/logs/"+requestId+".log", 200)
	if err != nil {
		fmt.Printf("Panic, %e\n", err)
	}
	if processesForTermination[requestId] {
		return
	}
	var done bool
	var doneCount int
	for {
		done = true
		doneCount = 0
		req, _ := http.NewRequest("GET", endpoint+"request_status/"+requestId, nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Add("api-key", apiKey)
		client := http.Client{Timeout: httpTimeoutDuration} //nolint:exhaustivestruct
		res, err := client.Do(req)
		if err != nil {
			if err.(*url.Error).Timeout() {
				logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " ", os.Stderr, "Error connection timeout for request_status <br>")
			} else {
				logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " ", os.Stderr, "Error creating request for request_status <br>")
			}
			return
		}
		if res.StatusCode != http.StatusOK {
			switch res.StatusCode {
			case 404:
				logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " ", os.Stderr, "Error for request_status server responded 404 - Request doesn't exist. <br>")
				return
			case 401:
				logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " ", os.Stderr, "Error for request_status server responded 422 - Request doesn't belong to this API key. <br>")
				return
			default:
				logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " ", os.Stderr, "Error for request_status server responded "+strconv.Itoa(res.StatusCode)+" (this error is not implemented) <br>")
				return
			}
		}

		var respData map[string]interface{}
		err = json.NewDecoder(res.Body).Decode(&respData)
		if err != nil {
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Error decoding status response <br>")
			return
		}
		res.Body.Close()
		client.CloseIdleConnections()
		for _, file := range files {
			var img map[string]interface{}
			if _, ok := respData["request_status"]; ok {
				img = respData["request_status"].(map[string]interface{})
				img = img[ocrFilename(file)].(map[string]interface{})
				if img["state"] == "PROCESSED" {
					doneCount += 1
					continue
				} else {
					done = false
				}
			}
		}
		if processesForTermination[requestId] {
			done = true
		} else if doneCount != len(files) {
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " OCRs are not done yet (", doneCount, " / ", len(files), "), trying again in 60 seconds... <br>\n")
			changeNumOfDoneFiles(requestId, doneCount)
			time.Sleep(60 * time.Second)
		} else {
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " OCRs are not done yet (", doneCount, " / ", len(files), "), trying again in 60 seconds... <br>\n")
			changeNumOfDoneFiles(requestId, doneCount)
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " OCRs are done! <br>\n")
		}

		if done {
			break
		}
	}
}

func mustOpen(f string) *os.File {
	r, err := os.Open(f)
	if err != nil {
		fmt.Printf("Panic while openning file, %e\n", err)
		return nil
	}
	return r
}

// convertTiffToJpg
//
// if tiff convert to jpg
// if something else, not modified path is returned
func convertTiffToJpg(file string, requestId string) string {
	logger, err := gologger.New("./templates/logs/"+requestId+".log", 200)
	if err != nil {
		fmt.Printf("Panic, %e\n", err)
	}
	ext := filepath.Ext(file)
	if strings.ToLower(ext) == ".tif" || strings.ToLower(ext) == ".tiff" {
		f, err := os.Open(file)
		if err != nil {
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Couldn't open file: "+file+" skipping... <br>")
			return ""
		}
		decode, err := tiff.Decode(f)
		if err != nil {
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Couldn't read TIFF file: "+file+" skipping... <br>")
			f.Close()
			return ""
		}
		jpgFile, err := os.Create(strings.TrimSuffix(file, ext) + ".jpg")
		if err != nil {
			f.Close()
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Cannot create output file skipping... <br>")
			return ""
		}
		err = jpeg.Encode(jpgFile, decode, nil)
		if err != nil {
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Error encoding file from decode to jpg! <br>")
			return ""
		}

		f.Close()
		jpgFile.Close()
		return strings.TrimSuffix(file, ext) + ".jpg"
	}
	return file
}

// Testing if file has supported filename extension
func tiffExtensionTest(extension string) bool {
	if extension == ".tiff" || extension == ".tif" {
		return true
	}
	return false
}
