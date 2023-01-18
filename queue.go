package main

import (
	"container/heap"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/syrinsecurity/gologger"
)

var (
	Images   = make(map[int]*Image)
	IP       = NewImagePreperation()
	SlowDown = 0
)

type ImagePreperation struct {
	processQueue ProcessQueue
}

func NewImagePreperation() *ImagePreperation {
	ip := &ImagePreperation{processQueue: make(ProcessQueue, 0)}
	heap.Init(&ip.processQueue)
	return ip
}

// Add new image for preperation
func (ip *ImagePreperation) AddImage(image interface{}) {
	imageForPrep := image.(*Image)
	heap.Push(&ip.processQueue, imageForPrep)
}

// Remove image with the highest priority
func (ip *ImagePreperation) HandleNextImage() *Image {
	if ip.processQueue.Len() == 0 {
		return nil
	}
	nextImage := heap.Pop(&ip.processQueue)
	return nextImage.(*Image)
}

// Get next image, does not pop, just peek
func (ip *ImagePreperation) GetNextImage() interface{} {
	if ip.processQueue.Len() == 0 {
		return nil
	}
	return ip.processQueue.Peek()
}

// Get next image, does not pop, just peek
func (ip *ImagePreperation) GetNextImageObject() *Image {
	if ip.processQueue.Len() == 0 {
		return nil
	}
	return ip.processQueue.PeekForObject()
}

// Get lenght of imprep Queue
func (ip *ImagePreperation) LenghtOfIpQueue() int {
	return ip.processQueue.Len()
}

// Update image's priority
func (ip *ImagePreperation) UpdateImagePriority(image *Image, newPriority int) {
	image.priority = newPriority
	heap.Fix(&ip.processQueue, image.index)
}

// Set how much files processed directory has
func (ip *ImagePreperation) SetNumberOfFiles(image *Image, numOfFiles int) {
	image.lenght = numOfFiles
}

// Update number of completed images in given process
func (ip *ImagePreperation) UpdateProcessState(image *Image, processId int) {
	image.state = image.state + 1
	db.Model(&Logs{}).Where("log_id = ?", strconv.Itoa(processId)).Update("state", strconv.Itoa(image.state)+"/"+strconv.Itoa(image.lenght))
}

func UpradeProcessPriority(processId int, newPriority int) {
	IP.UpdateImagePriority(Images[processId], newPriority)
}

// endless loop checks for next process every 5 seconds.
func ChceckForNextProcess() {
	definedNumOfProcesses, err := strconv.Atoi(os.Getenv("NUM_OF_IMPREP_PROCESSES"))
	if err != nil {
		fmt.Println("Predefined number of processes is configured wrong")
	}
	for {
		nextImage := IP.GetNextImageObject()
		if nextImage != nil && numOfProcesses < definedNumOfProcesses {
			if processesForTerminationImg[nextImage.WorkerID] {
				delete(Images, nextImage.WorkerID)
				IP.HandleNextImage()
				delete(processesForTerminationImg, nextImage.WorkerID)
				time.Sleep(time.Second * 2)
				continue
			}
			numOfProcesses++
			IMPREP(nextImage.Input, nextImage.Output, nextImage.WorkerID)
		} else {
			SlowDown++
		}
		if SlowDown > 720 {
			time.Sleep(time.Second * 30)
			SlowDown--
		} else {
			time.Sleep(time.Second * 5)
		}
	}
}

// recieves request from main class for image preparation
func POSTHandler(input string, output string, processType string) {

	//gets all processes from DB
	var loogs []Logs
	lastLog := db.Find(&loogs)

	//gets last process id from DB
	numOfProc := strconv.FormatInt(lastLog.RowsAffected+1, 10)

	process := &Logs{
		LogID:     numOfProc,
		Input:     input,
		Priority:  int(lastLog.RowsAffected + 1),
		ShowBtn:   true,
		TimeStart: "Not started yet",
		TimeEnd:   "Not ended yet",
	}

	switch processType {
	case "imgprep":
		process.Status = "Waiting"
		process.Type = "Imgprep"
	// case "pero":
	// 	process.Status = "Running"
	// 	process.Type = "Pero"
	case "soundprep":
		process.Status = "Waiting"
		process.Type = "Soundprep"
	}

	//create new procces for DB
	createdProcess := db.Create(&process)
	err = createdProcess.Error
	if err != nil {
		fmt.Println(err, "<br>")
	}

	numOfProcInt, err := strconv.Atoi(numOfProc)
	if err != nil {
		fmt.Println("Failed to add counter to new process <br>")
		return
	}

	//new instance of process
	newProcess := &Image{Input: input, Output: output, WorkerID: numOfProcInt, priority: int(lastLog.RowsAffected + 1)}

	//logging package setup
	logger, err := gologger.New("./templates/logs/"+strconv.Itoa(numOfProcInt)+".log", 200)
	if err != nil {
		fmt.Printf("Panic, %e\n", err)
	}
	logger.WriteString(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Logs for input ", input, "<br>")

	//add new process to queue
	Images[numOfProcInt] = newProcess
	IP.AddImage(newProcess)

}
