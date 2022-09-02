// Populate the examples.
(function () {
    let examples = [
        {
            description: "Site visits over time in the past year",
            jsonQuery: {
                Graph: ["Count", "Pageview"],
                GroupBy: ["Month"],
                DateRange: "Past12Months",
            },
        },
        {
            description: "Page viewed from the Firefox browser from italian clients in the last 7 days",
            jsonQuery: {
                Graph: ["Count", "Pageview"],
                GroupBy: ["Day"],
                Filters: [
                    {
                        "Column": "language",
                        "Comparison": "Equal",
                        "Target": "'it'"
                    },
                    {
                        "Column": "browser",
                        "Comparison": "Contains",
                        "Target": "firefox",
                    }
                ],
                DateRange: "Past7Days"
            },
        },
        {
            description: "Clicks from clients outside Italy today",
            jsonQuery: {
                Graph: ["Count", "Click"],
                GroupBy: ["Day"],
                Filters: [
                    {
                        "Column": "language",
                        "Comparison": "NotEqual",
                        "Target": "'it'"
                    }
                ],
                DateRange: "Past7Days",
            },
        },
        {
            description: "Page viewed from 2021/08/03 to 12:34 of 01/01/2022",
            jsonQuery: {
                Graph: ["Count", "Pageview"],
                GroupBy: ["Year"],
                DateFrom: "2021-08-03 00:00:00",
                DateTo: "2022-01-01 12:34:00"
            },
        },
        {
            description: "Unique visitors today",
            jsonQuery: {
                Graph: ["Count Unique", "Pageview"],
                DateRange: "Today"
            },
        },
        {
            description: "Unique visitors yesterday",
            jsonQuery: {
                Graph: ["Count Unique", "Pageview"],
                DateRange: "Yesterday"
            },
        },
        {
            description: "Unique visitors within last 7 days",
            jsonQuery: {
                Graph: ["Count Unique", "Pageview"],
                DateRange: "Past7Days",
                GroupBy: ["Day"],
            },
        },
        {
            description: "Count of unique clicks on Login button (by channel) within last 31 days",
            jsonQuery: {
                Graph: ["Count Unique", "Click"],
                Filters: [
                    {
                        "Column": "target",
                        "Comparison": "Contains",
                        "Target": "Login",
                    }
                ],
                DateRange: "Past31Days",
                GroupBy: ["referrer"],
            },
        },
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