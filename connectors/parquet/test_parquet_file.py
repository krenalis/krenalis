import argparse
import sys

import pandas as pd


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("file")
    args = parser.parse_args()

    parquet_filename = args.file

    df = pd.read_parquet(parquet_filename)

    try:
        assert len(df.dtypes) == 18, f"unexpected {len(df.dtypes)} columns"
        assert df.subscribed[0] == True, df.subscribed[0]
        assert df.first_name[0] == "John", f"unexpected: {df.first_name[0]}"
        assert df.last_name[0] == "Lemon", f"unexpected: {df.last_name[0]}"
    except AssertionError as ex:
        print(
            f"Python: the validation of the Parquet file failed as an assertion failed",
            file=sys.stderr,
        )
        raise ex

    print("Python: the Parquet file seems to be ok")


if __name__ == "__main__":
    main()
