export let docsData = `{
    "Graph":
            { "Count", "Pageview" | "Click" } |
            { "Count Unique", "Pageview" | "Click" },

    "Filters": [{
            "Column": <a column>,
            "Comparison": "Equal" | "NotEqual" | "GreaterThan" 
            | "GreaterThanEqual" | "LessThan" | "LessThanEqual" 
            | "Contains", "NotContains",
            "Target": <a value>, 
    }, ...]

    "GroupBy": <a column> | "Day" | "Month" | "Year",

    "DateRange": "Today" | "Yesterday" | "Past7Days" 
    | "Past31Days" | "Past12Months",

    "DateFrom": <a timestamp>,
    "DateTo": <a timestamp>
}

columns = "timestamp" | "browser" | "language" | "referrer" 
| "session" | "target" | "text" | "title" | "url"

// Timestamp should have the form: YYYY-MM-DD HH:MM:SS`