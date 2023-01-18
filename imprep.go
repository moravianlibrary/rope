package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/syrinsecurity/gologger"
)

const (
	previewHeight = 1000
	thumbHeight   = 128
)

var (
	jp2Convert           = false
	proccesingNumImg int = 0

	numOfProcesses int = 0
	runningWorker      = make(map[int]struct{})

	processesForTerminationImg = make(map[int]bool)

	LastProcessWasNil  = true
	filesForConversion = false
)

func (ip *ImagePreperation) TerminateImgprepProcess(processId int) {
	processesForTerminationImg[processId] = true
	//if process is running, termanation will be managed elsewhere
	_, ok := runningWorker[processId]
	if !ok {
		proc := db.Where("id = ? AND status = ?", processId, "Running").First(&Logs{})
		//handles situation when app was closed with process running and after reboot, user wants to terminate the process
		if proc == nil {
			ip.processQueue.Terminate(Images[processId], processId)
		}

		logger, err := gologger.New("./templates/logs/"+strconv.Itoa(processId)+".log", 200)
		if err != nil {
			fmt.Printf("Panic, %e\n", err)
		}
		delete(Images, processId)
		db.Model(&Logs{}).Where("log_id = ?", strconv.Itoa(processId)).Update("status", "Terminated")
		db.Model(&Logs{}).Where("log_id = ?", strconv.Itoa(processId)).Update("show_btn", false)
		db.Model(&Logs{}).Where("log_id = ?", strconv.Itoa(processId)).Update("time_end", time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"))
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Process", processId, "has been terminated <br>\n")
	}
}

func IMPREP(input string, output string, workerID int) {
	runningWorker[workerID] = struct{}{}
	fmt.Println("Delka fronty je: ", IP.LenghtOfIpQueue())
	fmt.Println("Na vrhcolu je: ", IP.GetNextImage())

	IP.HandleNextImage()

	logger, err := gologger.New("./templates/logs/"+strconv.Itoa(workerID)+".log", 200)
	if err != nil {
		fmt.Printf("Panic, %e\n", err)
	}

	db.Model(&Logs{}).Where("log_id = ?", strconv.Itoa(workerID)).Update("status", "Running")
	db.Model(&Logs{}).Where("log_id = ?", strconv.Itoa(workerID)).Update("time_start", time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"))
	if _, err := os.Stat(input); os.IsNotExist(err) {
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Input path does not exist! <br>")
		numOfProcesses--
		setProcessAsFinished(workerID)
		return
	}
	if _, err := os.Stat(output); os.IsNotExist(err) {
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Output path does not exist <br>")
		err := os.MkdirAll(output, os.ModePerm)

		if err != nil {
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Cannot create output folder! <br>")
			numOfProcesses--
			setProcessAsFinished(workerID)
			return
		} else {
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Output folder has been created <br>")
		}
	}

	srcDir, err := os.Open(input)
	if err != nil {
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Cannot open input directory <br>")
		numOfProcesses--
		setProcessAsFinished(workerID)
		return
	}
	defer srcDir.Close()

	outDir, err := os.Open(output)
	if err != nil {
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Cannot open output directory <br>")
		numOfProcesses--
		setProcessAsFinished(workerID)
		return
	}
	defer outDir.Close()

	files, err := srcDir.ReadDir(0)
	if err != nil {
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Cannot read files in directory <br>")
		numOfProcesses--
		setProcessAsFinished(workerID)
		return
	}

	// check if opj_compress exists
	opj := ""
	if runtime.GOOS == "windows" {
		opj, err = exec.LookPath("opj_compress.exe")
		if err == nil {
			jp2Convert = true
		}
	} else {
		opj, err = exec.LookPath("opj_compress")
		if err == nil {
			jp2Convert = true
		}
	}

	IP.SetNumberOfFiles(Images[workerID], fileCount(input))

	for _, file := range files {

		if processesForTerminationImg[workerID] {
			setProcessAsFinished(workerID)
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Process ", workerID, " has been terminated <br>\n")
			delete(processesForTerminationImg, workerID)

			numOfProcesses--
			NextImg := IP.GetNextImage()
			println("nasleduje: ", NextImg)
			return
		}

		if file.IsDir() {
			fileRecursion(file.Name(), outDir, srcDir, opj, workerID)
			continue
		}
		//gets file name extension
		ext := filepath.Ext(file.Name())
		//supported filename extension: "jpg" (or "jpeg"), "png", "gif", "tif" (or "tiff") and "bmp" (others are not supported by imaging package)
		if fileExtensionTest(ext) {

			for proccesingNumImg >= 5 {
				time.Sleep(time.Second * 1)
			}
			imagePrep(outDir, srcDir, file.Name(), ext, opj, workerID)
			time.Sleep(time.Second * 5)
		} else {
			err := MoveFile(filepath.Join(srcDir.Name(), file.Name()), filepath.Join(outDir.Name(), file.Name()))
			if err != nil {
				logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " An error occurred with moving file", file.Name(), "<br>\n")
			}
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " ", file.Name(), " has unsupported file extension <br>\n")
		}

	}
	for proccesingNumImg > 1 {
		if processesForTerminationImg[workerID] {
			db.Model(&Logs{}).Where("log_id = ?", strconv.Itoa(workerID)).Update("status", "Terminated")
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Process ", workerID, " has been terminated <br>\n")
			delete(processesForTerminationImg, workerID)
		}
		time.Sleep(time.Second * 1)
	}
	db.Model(&Logs{}).Where("log_id = ?", strconv.Itoa(workerID)).Update("status", "Finishing")
	time.Sleep(time.Second * 60)

	numOfProcesses--

	err = os.Remove(srcDir.Name())
	if err != nil {
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), "Unable to remove directory <br>\n")
	}

	if processesForTerminationImg[workerID] {
		db.Model(&Logs{}).Where("log_id = ?", strconv.Itoa(workerID)).Update("status", "Terminated")
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Process ", workerID, " has been terminated <br>\n")
		delete(processesForTerminationImg, workerID)
	} else {
		db.Model(&Logs{}).Where("log_id = ?", strconv.Itoa(workerID)).Update("status", "Finished")
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), "Process is done.<br>\n")
	}
	db.Model(&Logs{}).Where("log_id = ?", strconv.Itoa(workerID)).Update("time_end", time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"))
	db.Model(&Logs{}).Where("log_id = ?", strconv.Itoa(workerID)).Update("show_btn", false)
	ImageToBeHandled := IP.GetNextImage()
	delete(Images, workerID)
	delete(runningWorker, workerID)
	println("Další proces je: ", ImageToBeHandled)
}

func fileRecursion(fileName string, outDir *os.File, srcDir *os.File, opj string, workerID int) {
	logger, err := gologger.New("./templates/logs/"+strconv.Itoa(workerID)+".log", 200)
	if err != nil {
		fmt.Printf("Panic, %e\n", err)
	}
	subDir, err := os.Open(filepath.Join(srcDir.Name(), fileName))
	if err != nil {
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Cannot open input directory <br>")
		numOfProcesses--
		return
	}
	defer subDir.Close()

	subFiles, err := subDir.ReadDir(0)
	if err != nil {
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Cannot read files in directory <br>")
		numOfProcesses--
		return
	}
	for _, subFile := range subFiles {
		if processesForTerminationImg[workerID] {
			numOfProcesses--
			return
		}
		if subFile.IsDir() {
			fileRecursion(subFile.Name(), outDir, subDir, opj, workerID)
			continue
		}
		ext := filepath.Ext(subFile.Name())
		if fileExtensionTest(ext) {
			for proccesingNumImg >= 5 {
				time.Sleep(time.Second * 1)
			}
			imagePrep(outDir, subDir, subFile.Name(), ext, opj, workerID)
			time.Sleep(time.Second * 1)
		} else {
			err := MoveFile(filepath.Join(srcDir.Name(), subFile.Name()), filepath.Join(outDir.Name(), subFile.Name()))
			if err != nil {
				fmt.Printf("Couldn't open source file: %s\n", err)
			}
			println("File je:", filepath.Join(srcDir.Name(), subFile.Name()))
			println("File je:", outDir.Name())
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " ", subFile.Name(), " has unsupported file extension <br>\n")
		}
	}
}

func decrement(workerID int) {
	if proccesingNumImg > 0 {
		proccesingNumImg--
	}
}

func imagePrep(outDir *os.File, srcDir *os.File, file string, ext string, opj string, workerID int) {
	if processesForTerminationImg[workerID] {
		return
	}
	proccesingNumImg++
	logger, err := gologger.New("./templates/logs/"+strconv.Itoa(workerID)+".log", 200)
	if err != nil {
		fmt.Printf("Panic, %e\n", err)
	}

	f, err := os.Open(filepath.Join(srcDir.Name(), file))
	if err != nil {
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Couldn't open file: "+file+" skipping... <br>")
		decrement(workerID)
		return
	}

	decodedImg, _, err := image.Decode(f)
	if err != nil {
		f.Close()
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Couldn't decode file: "+file+" skipping... <br>")
		decrement(workerID)
		return
	}

	jpgFile, err := os.Create(filepath.Join(outDir.Name(), strings.TrimSuffix(file, ext)+".full.jpg"))
	if err != nil {
		f.Close()
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Cannot create output file skipping... <br>")
		decrement(workerID)
		return

	}
	jpgPreviewFile, err := os.Create(filepath.Join(outDir.Name(), strings.TrimSuffix(file, ext)+".preview.jpg"))
	if err != nil {
		f.Close()
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Cannot create output file skipping... <br>")
		decrement(workerID)
		return

	}
	preview := imaging.Resize(decodedImg, previewHeight, 0, imaging.Lanczos)
	jpgThumbFile, err := os.Create(filepath.Join(outDir.Name(), strings.TrimSuffix(file, ext)+".thumb.jpg"))
	if err != nil {
		f.Close()
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Cannot create output file skipping... <br>")
		decrement(workerID)
		return

	}
	thumb := imaging.Resize(decodedImg, 0, thumbHeight, imaging.Lanczos)

	err = jpeg.Encode(jpgFile, decodedImg, &jpeg.Options{Quality: 90})
	if err != nil {
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Error encoding file from tiff to jpg! <br>")
		decrement(workerID)
		return
	}
	err = jpeg.Encode(jpgPreviewFile, preview, nil)
	if err != nil {
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Error encoding file from tiff to jpg preview! <br>")
		decrement(workerID)
		return
	}
	err = jpeg.Encode(jpgThumbFile, thumb, nil)
	if err != nil {
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Error encoding file from tiff to jpg thumb! <br>")
		decrement(workerID)
		return
	}

	f.Close()
	jpgFile.Close()
	jpgPreviewFile.Close()
	jpgThumbFile.Close()
	if jp2Convert {
		absSrc, _ := filepath.Abs(srcDir.Name())
		absOut, _ := filepath.Abs(outDir.Name())
		createJp2ImageArchivalCopy(workerID, opj, file, filepath.Join(absSrc, file), filepath.Join(absOut, strings.TrimSuffix(file, ext)+".NDK_ARCHIVAL.jp2"))
		createJp2ImageUserCopy(workerID, opj, file, filepath.Join(absSrc, file), filepath.Join(absOut, strings.TrimSuffix(file, ext)+".NDK_USER.jp2"))
	}

	if jp2Convert && filesForConversion {
		pngConvertedFile, err := os.Create(filepath.Join(srcDir.Name(), strings.TrimSuffix(file, ext)+".png"))
		if err != nil {
			f.Close()
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Cannot create output file skipping... <br>")

		}
		err = png.Encode(pngConvertedFile, decodedImg)
		if err != nil {
			logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Error converting file to png! <br>")
		}
		pngConvertedFile.Close()

		absSrc, _ := filepath.Abs(srcDir.Name())
		absOut, _ := filepath.Abs(outDir.Name())
		createJp2ImageArchivalCopy(workerID, opj, file, filepath.Join(absSrc, strings.TrimSuffix(file, ext)+".png"), filepath.Join(absOut, strings.TrimSuffix(file, ext)+".NDK_ARCHIVAL.jp2"))
		createJp2ImageUserCopy(workerID, opj, file, filepath.Join(absSrc, strings.TrimSuffix(file, ext)+".png"), filepath.Join(absOut, strings.TrimSuffix(file, ext)+".NDK_USER.jp2"))

		if _, err := os.Stat(filepath.Join(srcDir.Name(), strings.TrimSuffix(file, ext)+".png")); err == nil {
			e := os.Remove(filepath.Join(srcDir.Name(), strings.TrimSuffix(file, ext)+".png"))
			if e != nil {
				logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Errorek s prevodem formatu souboru")
			}
		}
	}
	filesForConversion = false

	logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " ✅ "+file+"<br>")
	IP.UpdateProcessState(Images[workerID], workerID)
	err = MoveFile(filepath.Join(srcDir.Name(), file), filepath.Join(outDir.Name(), file))
	if err != nil {
		fmt.Printf("Couldn't open source file: %s\n", err)
	}
	decrement(workerID)
}

func createJp2ImageArchivalCopy(workerID int, opj string, file string, filePath string, output string) {
	logger, error := gologger.New("./templates/logs/"+strconv.Itoa(workerID)+".log", 200)
	if error != nil {
		fmt.Printf("Panic, %e\n", err)
	}
	cmd := exec.Command(opj, "-i", filePath,
		"-o", output,
		"-b", "64,64",
		"-c", "[256,256],[256,256],[128,128]",
		"-t", "4096,4096",
		"-p", "RPCL",
		"-SOP",
		"-EPH",
		"-M", "1",
		"-TP", "R")
	_, err := cmd.CombinedOutput()
	if err != nil {
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Error running opj_compress for file '", file, "', converting file and trying again <br>")
		filesForConversion = true
	}
}

func createJp2ImageUserCopy(workerID int, opj string, file string, filePath string, output string) {
	logger, error := gologger.New("./templates/logs/"+strconv.Itoa(workerID)+".log", 200)
	if error != nil {
		fmt.Printf("Panic, %e\n", err)
	}
	cmd := exec.Command(opj, "-i", filePath,
		"-o", output,
		"-b", "64,64",
		"-c", "[256,256],[256,256],[128,128]",
		"-t", "1024,1024",
		"-p", "RPCL",
		"-SOP",
		"-EPH",
		"-M", "1",
		"-TP", "R",
		"-I")
	_, err := cmd.CombinedOutput()
	if err != nil {
		logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Error running opj_compress for file '", file, "', converting file and trying again <br>")
		filesForConversion = true
	}
}

// Testing if file has supported filename extension
func fileExtensionTest(extension string) bool {
	switch extension {
	case ".tif", ".tiff":
		return true
	case ".jpg", ".jpeg":
		return true
	case ".png":
		return true
	default:
		return false
	}
}

func setProcessAsFinished(workerID int) {
	db.Model(&Logs{}).Where("log_id = ?", strconv.Itoa(workerID)).Update("status", "Terminated")
	db.Model(&Logs{}).Where("log_id = ?", strconv.Itoa(workerID)).Update("time_end", time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"))
	db.Model(&Logs{}).Where("log_id = ?", strconv.Itoa(workerID)).Update("show_btn", false)
	delete(runningWorker, workerID)
	delete(Images, workerID)
}

func MoveFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("Couldn't open source file: %s", err)
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return fmt.Errorf("Couldn't open dest file: %s", err)
	}
	defer outputFile.Close()
	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		return fmt.Errorf("Writing to output file failed: %s", err)
	}
	// The copy was successful, so now delete the original file
	err = os.Remove(sourcePath)
	if err != nil {
		return fmt.Errorf("Failed removing original file: %s", err)
	}
	return nil
}

func fileCount(path string) int {
	i := 0
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return 0
	}
	for _, file := range files {
		ext := filepath.Ext(file.Name())
		if !file.IsDir() && fileExtensionTest(ext) {
			i++
			// } else if soundFileExtensionTest(ext) {
			// 	i++
			// }
		}
	}
	return i
}
