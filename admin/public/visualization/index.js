function updateResults() {

    const xhr = new XMLHttpRequest();
    const url = "https://localhost:9090/api/update-results"

    xhr.onload = function () {
        if (this.status != 200) {
            let error = xhr.getResponseHeader("X-Error");
            if (error == null) {
                switch (this.status) {
                    case 500:
                        error = "Internal Server Error"
                    case 400:
                        error = "Bad Request"    
                    default:
                        error = "Unknown server error"
                }
            }
            document.getElementById("updateResultsError").innerHTML = error;
            document.getElementById("updateResultsError").style.display = "block";
            return
        }
        document.getElementById("updateResultsError").innerHTML = "";
        document.getElementById("updateResultsError").style.display = "none";
        let result = JSON.parse(xhr.responseText);

        // Update the query.
        document.getElementById("query").innerHTML = xhr.getResponseHeader("X-Query");
        hljs.highlightAll();

        // Update the graph.
        (function () {
            let xValues = [];
            let yValues = [];
            for (let i = 0; i < result.length; i++) {
                xValues.push(result[i][0]);
                yValues.push(result[i][1]);
            }
            updateChart(xValues, yValues);
        })()

        // Create the table.
        let table = document.createElement("table");
        table.className = "results";
        let columnNames = xhr.getResponseHeader("X-Columns").split("|");
        let header = document.createElement("tr");
        for (let i = 0; i < columnNames.length; i++) {
            let th = document.createElement("th");
            th.innerHTML = columnNames[i];
            header.appendChild(th);
        }
        table.appendChild(header);
        for (let i = 0; i < result.length; i++) {
            let row = document.createElement("tr");
            for (j = 0; j < result[i].length; j++) {
                let cell = document.createElement("td");
                cell.innerHTML = result[i][j];
                row.appendChild(cell);
            }
            table.appendChild(row);
        }
        let resultNode = document.getElementById("result")
        if (resultNode.hasChildNodes()) {
            resultNode.removeChild(resultNode.firstChild);
        }
        resultNode.appendChild(table);
    }

    xhr.open("POST", url);
    let data = JSON.parse(editor.getValue());
    xhr.send(JSON.stringify(data));
}

function editorChange() {
    let autoUpdate = document.getElementById("autoUpdateResults").checked;
    if (autoUpdate) {
        updateResults();
    }
}