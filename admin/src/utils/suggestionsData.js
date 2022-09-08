export let suggestionsData = [
    {
        description: "Site visits over time in the past year",
        jsonQuery: `{
    "Graph": ["Count", "Pageview"],
    "GroupBy": ["Month"],
    "DateRange": "Past12Months"
}`,
    },
    {
        description: "Page viewed from the Firefox browser from italian clients in the last 7 days",
        jsonQuery: `{
    "Graph": ["Count", "Pageview"],
    "GroupBy": ["Day"],
    "Filters": [
        {
            "Field": "language",
            "Operator": "Equal",
            "": "'it'"
        },
        {
            "Field": "browserName",
            "Operator": "Contains",
            "": "firefox"
        }
    ],
    "DateRange": "Past7Days"
}`,
    },
    {
        description: "Clicks from clients outside Italy today",
        jsonQuery: `{
    "Graph": ["Count", "Click"],
    "GroupBy": ["Day"],
    "Filters": [
        {
            "Field": "language",
            "Operator": "NotEqual",
            "": "'it'"
        }
    ],
    "DateRange": "Past7Days"
}`,
    },
    {
        description: "Page viewed from 2021/08/03 to 12:34 of 01/01/2022",
        jsonQuery: `{
    "Graph": ["Count", "Pageview"],
    "GroupBy": ["Year"],
    "DateFrom": "2021-08-03 00:00:00",
    "DateTo": "2022-01-01 12:34:00"
}`,
    },
    {
        description: "Unique visitors today",
        jsonQuery: `{
    "Graph": ["Count Unique", "Pageview"],
    "DateRange": "Today"
}`,
    },
    {
        description: "Unique visitors yesterday",
        jsonQuery: `{
    "Graph": ["Count Unique", "Pageview"],
    "DateRange": "Yesterday"
}`,
    },
    {
        description: "Unique visitors within last 7 days",
        jsonQuery: `{
    "Graph": ["Count Unique", "Pageview"],
    "DateRange": "Past7Days",
    "GroupBy": ["Day"]
}`,
    },
    {
        description: "Count of unique clicks on Login button (by channel) within last 31 days",
        jsonQuery: `{
    "Graph": ["Count Unique", "Click"],
    "Filters": [
        {
            "Field": "target",
            "Operator": "Contains",
            "": "Login"
        }
    ],
    "DateRange": "Past31Days",
    "GroupBy": ["referrer"]
}`,
    },
    {
        description: "Count of unique clicks on Login button (by channel) within last 31 days, using a Smart Event",
        jsonQuery: JSON.stringify({
            "Graph": [
                "Count Unique",
                "Smart Event",
                "Click on Login Button"
            ],
            "DateRange": "Past31Days",
            "GroupBy": [
                "referrer"
            ]
        }, null, 2),
    },
    {
        description: "Most viewed Nissan Cars in the past 12 months",
        jsonQuery: JSON.stringify({
            "Graph": [
                "Count",
                "Smart Event",
                "View Nissan Car"
            ],
            "DateRange": "Past12Months",
            "GroupBy": [
                "url"
            ]
        }, null, 2),
    },
];
