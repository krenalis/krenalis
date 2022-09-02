export let suggestionsData = [
    {
        description: 'Site visits over time in the past year',
        query: {
            "GraphOn": "PageView",
            "GroupBy": "Month",
            "DateRange": "Past12Months"
        }
    },
    {
        description: 'Page viewed from italian clients in the last 7 days, grouped by URL',
        query: {
            "GraphOn": "PageView",
            "Filters": [
                {
                    "Column": "language",
                    "Comparison": "Equal",
                    "Target": "'it'"
                }
            ],
            "GroupBy": "url",
            "DateRange": "Past7Days"
        },
    },
    {
        description: 'Clicks from clients outside Italy today, grouped by URL',
        query: {
            "GraphOn": "Click",
            "Filters": [
                {
                    "Column": "language",
                    "Comparison": "NotEqual",
                    "Target": "'it'"
                }
            ],
            "GroupBy": "url",
            "DateRange": "Today"
        },
    }
]
