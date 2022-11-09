
## How to test on Windows

1. Download [Rebex Tiny SFTP Server](https://www.rebex.net/tiny-sftp-server/), or get from [GitHub](https://github.com/rebexnet/RebexTinySftpServer).
2. Unzip the file into a new directory.
3. Execute `RebexTinySftpServer.exe`.
4. Add the following setting to the `settings` column of the `Storage` data source: `{"Host":"192.168.1.2","Port":2222,"Username":"tester","Password":"password","Path":"/users.csv"}`
5. Change the IP with your IP (It also appears in the Rebex Tiny SFTP server window).
6. Places the file `users.csv` to be served in the `data` directory of the Rebex directory.
