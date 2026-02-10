# go-cblgrabber

Data grabber for ČBL (Czech Basketball League) from [https.//nbl.basketball](https.//nbl.basketball/zapasy). All data are stored in [SQLite](https://www.sqlite.org/) database which is used also in other projects.

## Usage

```bash
$ ./go-cblgrabber -help

Usage of ./go-cblgrabber:
-database string
        Path to the database (default "./data.db")
-initdb
        Initialize database - existing data will be erased
-season string
        Season we want to grab (default "2020/21")

# 1. starting with new database
$ ./go-cblgrabbeer -initdb -season=2022/23

Grabbing URL: https://nbl.basketball/zapasy?y=2022&p1=0&c=0&d_od=&d_do=&k=0

# 2. continue grabbing
$ ./go-cblgrabber -season=2023/24

Grabbing URL: https://nbl.basketball/zapasy?y=2023&p1=0&c=0&d_od=&d_do=&k=0

# 3. check database
$ ls data.db -l

.rw-r--r-- 106k ondrejd 10 úno 17:00 data.db

# 4. open database
$ sqlite3 data.db

SQLite version 3.46.1 2024-08-13 09:16:08
Enter ".help" for usage hints.
sqlite> SELECT id, name FROM teams;
1|NH Ostrava
2|BK KVIS Pardubice
3|USK Praha
4|Královští sokoli
5|PUMPA Basket Brno
6|SK Slavia Praha
7|BK Olomoucko
8|BC GEOSAN Kolín
9|BK Opava
10|ERA Basketball Nymburk
11|SLUNETA Ústí nad Labem
12|BK ARMEX ENERGY Děčín
13|Sršni Photomate Písek
sqlite> 

```

