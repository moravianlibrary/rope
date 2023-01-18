// Restricts input for the given textbox to the given inputFilter.
function setInputFilter(textbox, inputFilter, errMsg) {
    ["input", "keydown", "keyup", "mousedown", "mouseup", "select", "contextmenu", "drop", "focusout"].forEach(function(event) {
      textbox.addEventListener(event, function(e) {
        if (inputFilter(this.value)) {
          // Accepted value
          if (["keydown", "mousedown", "focusout"].indexOf(e.type) >= 0) {
            this.classList.remove("input-error");
            this.setCustomValidity("");
          }
          this.oldValue = this.value;
          this.oldSelectionStart = this.selectionStart;
          this.oldSelectionEnd = this.selectionEnd;
        } else if (this.hasOwnProperty("oldValue")) {
          // Rejected value - restore the previous one
          this.classList.add("input-error");
          this.setCustomValidity(errMsg);
          this.reportValidity();
          this.value = this.oldValue;
          this.setSelectionRange(this.oldSelectionStart, this.oldSelectionEnd);
        } else {
          // Rejected value - nothing to restore
          this.value = "";
        }
      });
    });
  }
  
  
  // Install input filters.
  setInputFilter(document.getElementById("engineId"), function(value) {
    return /^[1-6]*$/.test(value);
  }, "Must be an unsigned integer 1-6");

  // document.getElementById("fileInput").addEventListener("change", function(event) {
  //   let output =  document.getElementById("filename");
  //   let files = event.target.files;
  //   output.value = "test";
  //   console.log("Testuju");
  //   console.log(files[0].webkitRelativePath);
  // }, false);

// Restricts input for the given textbox to the given inputFilter.
function setInputFilter(textbox, inputFilter, errMsg) {
  ["input", "keydown", "keyup", "mousedown", "mouseup", "select", "contextmenu", "drop", "focusout"].forEach(function (event) {
    textbox.addEventListener(event, function (e) {
      if (inputFilter(this.value)) {
        // Accepted value
        if (["keydown", "mousedown", "focusout"].indexOf(e.type) >= 0) {
          this.classList.remove("input-error");
          this.setCustomValidity("");
        }
        this.oldValue = this.value;
        this.oldSelectionStart = this.selectionStart;
        this.oldSelectionEnd = this.selectionEnd;
      } else if (this.hasOwnProperty("oldValue")) {
        // Rejected value - restore the previous one
        this.classList.add("input-error");
        this.setCustomValidity(errMsg);
        this.reportValidity();
        this.value = this.oldValue;
        this.setSelectionRange(this.oldSelectionStart, this.oldSelectionEnd);
      } else {
        // Rejected value - nothing to restore
        this.value = "";
      }
    });
  });
}

function selectFolder(e) {
  let output = document.getElementById("filename");
  var theFiles = e.target.files;
  var relativePath = theFiles[0].webkitRelativePath;
  var folder = relativePath.split("/");
  folder[0] = "./" + folder[0];
  console.log(folder[0])
  output.value = folder[0];
}

function openPopup(){
  popup.classList.add("open-popup")
  pero.classList.add("hide")
}

function closePopup(){
  popup.classList.remove("open-popup")
  pero.classList.remove("hide")
}

function selectInput(path){
  document.getElementById("filename").value = path;
}

// process.tmpl
function openLog(logID){
  var xhr = new XMLHttpRequest();
    xhr.onreadystatechange = function () {
    }
    xhr.open('get', '/logs?logid=' + logID + '.log', true);
    xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded; charset=UTF-8');
    xhr.send();
}

function terminateProcess(logID, type){
  var xhr = new XMLHttpRequest();
    xhr.onreadystatechange = function () {
      if (xhr.readyState === 4) {
            alert("Ruším proces " + logID + ". Zruší se do 60sec.");
        }
    }
    if (type === "Imgprep"){
      xhr.open('get', '/terminateImg?logid=' + logID, true);
    } else {
      xhr.open('get', '/terminatePero?logid=' + logID, true);
    }
    xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded; charset=UTF-8');
    xhr.send();
}


// Install input filters.
document.getElementsByClassName("priorityInput").addEventListener("input", allowOnlyDigits);

function allowOnlyDigits() {  
console.log("Ano")
if (this.validity.valid) {
this.setAttribute('current-value', this.value.replace(/[^\d]/g, ""));
}
this.value = this.getAttribute('current-value');
}

function changePriority(logID, processType) {
let newPriority = document.getElementById("priorityInput"+logID).value;
console.log("Menim prioritu u:" + logID + " na:" + newPriority);
var xhr = new XMLHttpRequest();
    xhr.onreadystatechange = function () {
    }
    xhr.open('get', '/changePriority?logid=' + logID + '&type=' + processType + "&newPriority=" + newPriority, true);
    xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded; charset=UTF-8');
    xhr.send();
}