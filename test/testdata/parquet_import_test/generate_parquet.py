#
# SPDX-License-Identifier: Elastic-2.0
#
#
# Copyright (c) 2025 Open2b
#

from datetime import date, datetime, time, timezone
from decimal import Decimal

import pandas as pd

pd.DataFrame(
    [
        {"parquet_id": 100, "first_name": "John", "last_name": "Lemon"},
        {"parquet_id": 101, "first_name": "Ringo", "last_name": "Planett"},
        {"parquet_id": 102, "phone_numbers": ["111 222", "333 444"]},
        {"parquet_id": 103, "address": {"street": "ABC1", "zip_code": 1234}},
        {"parquet_id": 104, "date_of_birth": date(1980, 1, 2)},  # after 1970-01-01
        {"parquet_id": 105, "date_of_birth": date(1935, 1, 2)},  # before 1970-01-01
        {
            "parquet_id": 106,
            "updated_at": datetime(2012, 1, 20, 7, 20, 1, tzinfo=timezone.utc),
        },
        {"parquet_id": 107, "lunch_time": time(13, 30, 0)},
        {"parquet_id": 108, "score": Decimal("-1234.56789")},
    ]
).to_parquet("test.parquet")
