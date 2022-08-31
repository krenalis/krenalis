// Populate the examples.
(function () {
    let examples = [
        {
            description: "Site visits over time in the past year",
            jsonQuery: {
                GraphOn: "PageView",
                GroupBy: "Month",
                DateRange: "Past12Months"
            },
        },
        {
            description: "Page viewed from italian clients in the last 7 days, grouped by URL",
            jsonQuery: {
                GraphOn: "PageView",
                Filters: [
                    {
                        "Column": "language",
                        "Comparison": "Equal",
                        "Target": "'it'"
                    }
                ],
                GroupBy: "url",
                DateRange: "Past7Days"
            },
        },
        {
            description: "Clicks from clients outside Italy today, grouped by URL",
            jsonQuery: {
                GraphOn: "Click",
                Filters: [
                    {
                        "Column": "language",
                        "Comparison": "NotEqual",
                        "Target": "'it'"
                    }
                ],
                GroupBy: "url",
                DateRange: "Today"
            },
        }
    ];
    let examplesNode = document.getElementById("examples")
    examplesNode.innerHTML = "";
    let examplesTable = document.createElement("table");
    for (let i = 0; i < examples.length; i++) {
        let tr = document.createElement("tr");

        // First td -- description.
        let td1 = document.createElement("td")
        let span = document.createElement("span");
        span.innerHTML = examples[i].description;
        td1.appendChild(span);

        // Second td -- load button.
        let td2 = document.createElement("td");
        let btn = document.createElement("button");
        btn.innerHTML = "Load and update results";
        btn.onclick = function () {
            editor.setValue(JSON.stringify(examples[i].jsonQuery, null, 2));
            updateResults();
        }
        td2.appendChild(btn);

        tr.appendChild(td1);
        tr.appendChild(td2);
        examplesTable.appendChild(tr);
    }
    examplesNode.appendChild(examplesTable);
})()