package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB
var err error
var filesLoaded bool = false
var Files []File
var deepestNest int
var sizeOfNesting []int

type Logs struct {
	gorm.Model `gorm:"primaryKey"`

	LogID     string
	Input     string
	Status    string
	Type      string
	TimeStart string
	TimeEnd   string
	Priority  int
	ShowBtn   bool
	EngineID  int
	State     string //current state of process (current/max)
}

type File struct {
	Index       int
	NestedLevel int
	Path        string
	FolderName  string
	SubFolders  []File
}
type FORMACTION struct {
	FileName  string `form:"filename"`
	ProcessID string `form:"processIDinput"`
	EngineId  int    `form:"engineId"`
	Pull_only string `form:"pull_only"`
}

func main() {

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", os.Getenv("DB_HOST"), "rope", os.Getenv("DB_PASSWORD"), os.Getenv("DB_DB"), os.Getenv("DB_PORT"))
	db, err = gorm.Open(postgres.Open(dsn))

	if err != nil {
		fmt.Printf(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Panic, %e\n", err)
	} else {
		fmt.Println(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Connected to database successfully")
	}

	db.AutoMigrate(&Logs{})

	db.Model(&Logs{}).Where("status = ?", "Running").Update("status", "Crashed")
	db.Model(&Logs{}).Where("status = ?", "Crashed").Update("show_btn", "f")

	//run a loop that chcecks for a new process every 5 seconds
	go ChceckForNextProcess()

	router := gin.Default()
	router.LoadHTMLGlob("templates/**/*")

	router.GET("/", homePage)
	router.GET("/pero", peroPage)
	router.GET("/processes", processesPage)
	router.GET("/sound", soundPage)
	router.GET("/logs", getLog)

	router.GET("/terminateImg", terminateImgprep)
	router.GET("/terminatePero", terminatePero)

	router.GET("redoImgprep", redoImgprep)
	router.GET("redoPero", redoPero)
	router.GET("/changePriority", updatePriority)

	router.Static("/static", "./static/")
	router.Static("/src", "./src/")
	router.Static("/js", "./js/")

	router.POST("/imgprepaction", imgPrepAction)
	router.POST("/peroaction", peroAction)
	router.POST("/soundaction", convertSoundAction)

	router.POST("/perocancel", cancelPero)

	router.POST("/loadFoldersImg", loadFoldersImg)
	router.POST("/loadFoldersPero", loadFoldersPero)

	router.Run(":8080")
}

func homePage(c *gin.Context) {
	if filesLoaded {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"files": Files,
		})
	} else {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{})
	}
}

func peroPage(c *gin.Context) {
	if filesLoaded {
		c.HTML(http.StatusOK, "pero.tmpl", gin.H{
			"files": Files,
			// "ShowPopup": "open-popup",
			"hide":    "hide",
			"nesting": sizeOfNesting,
		})
	} else {
		c.HTML(http.StatusOK, "pero.tmpl", gin.H{})
	}
}

func processesPage(c *gin.Context) {
	var proc []Logs
	db.Order("id desc").Find(&proc)
	db.Order("id desc").Find(&proc)
	c.HTML(http.StatusOK, "processes.tmpl", gin.H{
		"proc": proc,
	})

}

func soundPage(c *gin.Context) {
	if filesLoaded {
		c.HTML(http.StatusOK, "sound.tmpl", gin.H{
			"files": Files,
		})
	} else {
		c.HTML(http.StatusOK, "sound.tmpl", gin.H{})
	}
}

func loadFoldersImg(c *gin.Context) {
	deepestNest = 0
	files := listDirectories("", 0, 0)
	filesLoaded = true
	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"files":     files,
		"ShowPopup": "open-popup",
		"hide":      "hide",
	})
}

func loadFoldersPero(c *gin.Context) {
	deepestNest = 0
	sizeOfNesting = nil
	files := listDirectories("", 0, 0)
	for i := 0; i < deepestNest; i++ {
		sizeOfNesting = append(sizeOfNesting, i)
	}
	filesLoaded = true
	c.HTML(http.StatusOK, "pero.tmpl", gin.H{
		"files":     files,
		"ShowPopup": "open-popup",
		"hide":      "hide",
		"nesting":   sizeOfNesting,
	})
}

func loadFoldersSound(c *gin.Context) {
	deepestNest = 0
	files := listDirectories("", 0, 0)
	filesLoaded = true
	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"files":     files,
		"ShowPopup": "open-popup",
		"hide":      "hide",
	})
}

func imgPrepAction(c *gin.Context) {
	var formAction FORMACTION
	c.Bind(&formAction)
	if filesLoaded {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"files": Files,
		})
	} else {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{})
	}
	formAction.FileName = modifyInput(formAction.FileName)
	for _, path := range findSubfolders(formAction.FileName) {
		go POSTHandler(path, makeOutputAddress(path), "imgprep")
		time.Sleep(time.Second * 1)
	}
}

func peroAction(c *gin.Context) {
	var formAction FORMACTION
	c.Bind(&formAction)
	if filesLoaded {
		c.HTML(http.StatusOK, "pero.tmpl", gin.H{
			"files": Files,
		})
	} else {
		c.HTML(http.StatusOK, "pero.tmpl", gin.H{})
	}
	if formAction.Pull_only == "true" {
		fmt.Println(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Pull-only has been activated")
	}
	pull_only, _ := strconv.ParseBool(formAction.Pull_only)
	formAction.FileName = modifyInput(formAction.FileName)

	for _, path := range findSubfolders(formAction.FileName) {
		go RunPero(path, path, formAction.EngineId, pull_only)
		time.Sleep(time.Second * 1)
	}

}

func convertSoundAction(c *gin.Context) {
	var formAction FORMACTION
	c.Bind(&formAction)
	if filesLoaded {
		c.HTML(http.StatusOK, "sound.tmpl", gin.H{
			"files": Files,
		})
	} else {
		c.HTML(http.StatusOK, "sound.tmpl", gin.H{})
	}
	formAction.FileName = modifyInput(formAction.FileName)
	for _, path := range findSubfolders(formAction.FileName) {
		go POSTHandler(path, makeOutputAddress(path), "soundprep")
	}
}

func cancelPero(c *gin.Context) {
	var formAction FORMACTION
	c.Bind(&formAction)
	go CancelOcrRequest(formAction.ProcessID)
	if filesLoaded {
		c.HTML(http.StatusOK, "pero.tmpl", gin.H{
			"files": Files,
		})
	} else {
		c.HTML(http.StatusOK, "pero.tmpl", gin.H{})
	}
}

func getLog(c *gin.Context) {
	logid := c.Query("logid")
	c.HTML(http.StatusOK, logid, gin.H{})
}

func terminateImgprep(c *gin.Context) {
	logid, err := strconv.Atoi(c.Query("logid"))
	fmt.Println(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Rusim img proces: ", logid)
	if err == nil {
		go IP.TerminateImgprepProcess(logid)
	}
}

func terminatePero(c *gin.Context) {
	logid := c.Query("logid")
	fmt.Println(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Rusim pero proces: ", logid)
	go CancelOcrRequest(logid)
}

func redoImgprep(c *gin.Context) {
	input := c.Query("input")
	if filesLoaded {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"files": Files,
		})
	} else {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{})
	}
	go POSTHandler(input, makeOutputAddress(input), "imgprep")
}

func redoPero(c *gin.Context) {
	input := c.Query("input")
	engineId, _ := strconv.Atoi(c.Query("engineId"))
	println("input je: " + input + ", engineId je: " + strconv.Itoa(engineId))
	if filesLoaded {
		c.HTML(http.StatusOK, "pero.tmpl", gin.H{
			"files": Files,
		})
	} else {
		c.HTML(http.StatusOK, "pero.tmpl", gin.H{})
	}
	go RunPero(input, input, engineId, false)

}

func updatePriority(c *gin.Context) {
	logid, err1 := strconv.Atoi(c.Query("logid"))
	newPriority, err2 := strconv.Atoi(c.Query("newPriority"))
	typeOfProcess := c.Query("type")

	if newPriority != 0 {
		db.Model(&Logs{}).Where("log_id = ?", c.Query("logid")).Update("priority", newPriority)

		fmt.Println(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " Changing priority on process:", logid, ", which is type ", typeOfProcess, " to new priority:", newPriority)
		if err1 == nil && err2 == nil && typeOfProcess == "Imgprep" {
			go UpradeProcessPriority(logid, newPriority)
		}
	}

}

func modifyInput(input string) string {
	if len(input) == 0 {
		return ""
	}
	return "./input/" + input
}

func makeOutputAddress(output string) string {
	if len(output) == 0 {
		return ""
	}
	lastInd := strings.LastIndex(output, "/")
	// return output[:lastInd] + "/hotovo/" + output[lastInd+1:]

	splited := strings.Split(output, "/")
	result := splited[0] + "/" + splited[1] + "/" + splited[2] + "/" + splited[3] + "/" + splited[4] + "/hotovo/" + output[lastInd+1:]
	return result
}

func listDirectories(input string, ind int, nested int) []File {
	if deepestNest < nested {
		deepestNest = nested
	}

	file, err := os.Open("./input" + input)
	if err != nil {
		file, err = os.Open("./input/" + input)
		if err != nil {
			fmt.Printf(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " failed opening directory: %s", err)
		}
	}
	defer file.Close()

	list, _ := file.Readdirnames(0)
	var files = []File{}
	for ind, s := range list {
		//check if path leaads to directory
		fileInfo, err := os.Stat("./input" + input + "/" + s)
		if err != nil {
			fileInfo, err = os.Stat("./input/" + input + "/" + s)
			if err != nil {
				fmt.Printf(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " failed checking directory: %s", err)
			}
		}
		if !fileInfo.IsDir() {
			continue
		}

		var folder File

		if input == "" {
			folder = File{
				Index:       ind,
				NestedLevel: nested,
				Path:        s,
				FolderName:  s,
				SubFolders:  listDirectories(s, ind, nested+1),
			}
		} else {
			folder = File{
				Index:       ind,
				NestedLevel: nested,
				Path:        input + "/" + s,
				FolderName:  s,
				SubFolders:  listDirectories(input+"/"+s, ind, nested+1),
			}
		}

		files = append(files, folder)

	}
	Files = files
	return files
}

func findSubfolders(inputPath string) []string {
	var IsFolderFinal = make(map[string]bool)
	var filesToBeProcessed []string
	isThereAtlOneDir := false
	file, err := os.Open(inputPath)
	if err != nil {
		fmt.Printf(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " failed opening directory: %s", err)
		//TODO dodelat funkci na popup a poslat to do FE
		filesToBeProcessed = append(filesToBeProcessed, inputPath)
		return filesToBeProcessed
	}

	defer file.Close()

	list, _ := file.Readdirnames(0)

	for _, file := range list {
		fileInfo, err := os.Stat(inputPath + "/" + file)
		if err != nil {
			fmt.Printf(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " failed checking directory: %s", err)
		}
		//test if fould file is a dir
		if fileInfo.IsDir() {
			IsFolderFinal[inputPath] = false
			isThereAtlOneDir = true
			testFiles := findSubfolders(inputPath + "/" + file)
			if testFiles[0] == inputPath+"/"+file {
				IsFolderFinal[inputPath+"/"+file] = true
			}
		} else {
			//if dir is already in map, do not overwrite it
			if _, ok := IsFolderFinal[inputPath]; !ok {
				IsFolderFinal[inputPath] = true
			}

		}

	}
	if !isThereAtlOneDir {
		filesToBeProcessed = append(filesToBeProcessed, inputPath)
		return filesToBeProcessed
	}

	for k, v := range IsFolderFinal {
		if v {
			println(time.Now().Add(time.Hour).Format("01/02/2006 15:04:05"), " nastavuji "+k+" jako to bo runned")
			filesToBeProcessed = append(filesToBeProcessed, k)
		}
	}
	return filesToBeProcessed
}
