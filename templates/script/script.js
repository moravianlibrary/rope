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

  document.getElementById("fileInput").addEventListener("change", function(event) {
    let output =  document.getElementById("filename");
    let files = event.target.files;
    output.value = "test";
    console.log("Testuju");
    console.log(files[0].webkitRelativePath);
  }, false);

  document.getElementById("chosenFile").addEventListener("click", myFunction);

  function myFunction() {
    console.log("Testuju");
  }