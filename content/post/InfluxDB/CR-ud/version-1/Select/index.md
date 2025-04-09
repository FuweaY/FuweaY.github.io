---
title: InfluxDB 1.X SELECT 語句
description: 介紹 InfluxDB 1.X SELECT 語句
slug: influxdb-1-CR-ud/select
date: 2021-01-22T12:00:00+08:00
categories:
   - InfluxDB
weight: 1  
---
InfluxQL 是 SQL LIKE 語法，所以整體而言的語法很 SQL 沒有太大的區別，以下文件假定使用者了解SQL語法，因此只簡列出特別需要注意的部分。以下為目錄：

## SELECT ... FROM ...

用於從 `FROM` 指定的 `measurement`，將 `SELECT` 的 `fields` 和 `tags` 顯示出來。

```sql
SELECT <field_key>[,<field_key>,<tag_key>]
FROM <measurement_name>[,<measurement_name>]
```

- `SELECT *` 表示返回所有的 `field_key` `tag_key`。
- 當需要 `SELECT tag_key` 時，至少需要包含一個 `field_key`，否則不會返回任何結果，這是有關於數據儲存方式的結果。
- `SELECT "<field_key>"::field,"<tag_key>"::tag` 其中 `::[field | tag]` 是用來區分擁有相同名稱的 `field_key` 和 `tag_key`，當同名沒有指定 `::[field | tag]` 時，預設會是顯示 `field key`

  範例：

    ```sql
    > INSERT test,host=A host="B",loading=0.5
    > SELECT * FROM test
    name: test
    time                host host_1 loading
    ----                ---- ------ -------
    1607668578356932984 B    A      0.5
    > SELECT host,loading FROM test
    name: test
    time                host loading
    ----                ---- -------
    1607668578356932984 B    0.5
    > SELECT host::tag,loading FROM test
    name: test
    time                host loading
    ----                ---- -------
    1607668578356932984 A    0.5
    ```

- `SELECT` 不可在使用 `聚合函數` 時，包含 `非聚合函數`、`field key` 或 `tag key`。也就是相當於 MySQL 中 sql_mode 包含 `ONLY_FULL_GROUP_BY` 的情況。

    ```sql
    > SELECT * FROM test
    name: test
    time                host loading
    ----                ---- -------
    1607672144573710142 A    0.5
    1607672146554038115 A    0.6
    > SELECT host,SUM(loading) FROM test
    ERR: mixing aggregate and non-aggregate queries is not supported 
    ```

- `FROM <database_name>.<retention_policy_name>.<measurement_name>` 可以使用此方式指定到特定 `database` 和 `retention policy` 的 `measurement`。
- `FROM <database_name>..<measurement_name>` 此方式會指定到對應 `database` 使用 `DEFAULT` RP 的 `measuremen`。
- `FROM <measurement_name>,<measurement_name>` 會傳回多個 `measurement` 的數據。

    ```sql
    > INSERT test,host=A loading=0.5,extra="test"
    > INSERT test2,host=B loading=0.5,extra2="test2"
    > SELECT * FROM test,test2
    name: test
    time                extra extra2 host loading
    ----                ----- ------ ---- -------
    1607672996339840129 test         A    0.5
    
    name: test2
    time                extra extra2 host loading
    ----                ----- ------ ---- -------
    1607673009199242222       test2  B    0.5
    ```

- 引號的規則和 [line protocol](INSERT%204b63194700ec4ef3b76f946cde1ef0ae.md) 不同，規則如下：
    1. `'` 用於 `String` 和 `timestamp` values，請勿用於標識符(database names, retention policy names, user names, measurement names, tag keys, and field keys)。
    2. `"` 用於標識符，標識符database names, retention policy names, user names, measurement names, tag keys, and field keys)。

    ```sql
    -- YES
    SELECT bikes_available FROM bikes WHERE station_id='9'
    SELECT "bikes_available" FROM "bikes" WHERE "station_id"='9'
    SELECT MIN("avgrq-sz") AS "min_avgrq-sz" FROM telegraf
    SELECT * from "cr@zy" where "p^e"='2'
    SELECT "water_level" FROM "h2o_feet" WHERE time > '2015-08-18T23:00:01.232000000Z' AND time < '2015-09-19'
    -- NO
    SELECT 'bikes_available' FROM 'bikes' WHERE 'station_id'="9"
    SELECT * from cr@zy where p^e='2'
    SELECT "water_level" FROM "h2o_feet" WHERE time > "2015-08-18T23:00:01.232000000Z" AND time < "2015-09-19"
    ```


## WHERE

只顯示符合 `WHERE` 條件的資料。

```sql
SELECT ... FROM ... WHERE <conditional_expression> [(AND|OR) <conditional_expression> [...]]
```

### Fields

```sql
field_key <operator> ['string' | boolean | float | integer]
```

- `String field value` 必須使用 `'` 包起來，否則不會返回數據，並起大部分時候不會返回錯誤。
- operator 支援：`=`、 `<>`、`!=`、`>`、`>=`、`<`、`<=`、算術運算符和正則表達式。

### Tags

```sql
tag_key <operator> ['tag_value']
```

- `tag value` 必須使用 `'` 包起來，否則不會返回數據，並起大部分時候不會返回錯誤。
- operator 支援：`=`、 `<>`、`!=` 和 正則表達式。

### Timestamp

- 預設值為 (UTC) `1677-09-21 00:12:43.145224194` ~ `2262-04-11T23:47:16.854775806Z`
- 當包含 `GROUP BY time()` 時，預設值為 (UTC) `1677-09-21 00:12:43.145224194` ~ `now()`
- 不支持在 `time` 上使用 `OR` 指定多個時間範圍，對於該查詢會返回空結果。

## SELECT INTO

將 `SELECT` 的結果寫入到 `INTO` 指定的 `measurement`

```sql
SELECT_clause INTO <measurement_name> FROM_clause [WHERE_clause] [GROUP_BY_clause]
```

- 可用於 `RENAME DATABASE`，如下範例：

    ```sql
    SELECT * INTO "copy_test"."autogen".:MEASUREMENT FROM "test"."autogen"./.*/ GROUP BY *
    ```

    1. 將 `test` DB 下 `autogen` RP 的資料寫入到 `copy_test` DB 下 `autogen` RP 。
    2. 反向引用語法 `<:MEASUREMENT>` 讓 `copy_test` DB 中 `measurement` 名稱引用 `test` DB 中的名稱。請注意在執行前必須確保 `copy_test` DB 及 `autogen` RP 存在。
    3. 若語法中未包含 `GROUP BY *` 在新的 `measurement` 中 `tag` 會被轉為 `field`。
    4. 當移動大量數據時，建議在 `WHERE` 條件中加入時間條件分批寫入，避免系統內存不足。
- 經常用於採樣，將高精度的資料透過 `function` 和 `group by` 聚合成低精度的數據。
- 由於當語法中未包含 `GROUP BY *` 時，原表的 `tag` 在新表將被轉換為 `field`，因此原本用 `tag` 區別的數據被 `overwrite`，這可能導致數據的丟失。若要保留 `tag` 請務必加上 `GROUP BY *`。

## GROUP BY

將數據依照 `GROUP BY` 指定的 `tag key` 或 `時間區間` 進行分組並聚合。

### tags

```sql
SELECT_clause FROM_clause [WHERE_clause] GROUP BY [* | <tag_key>[,<tag_key]]
```

- `GROUP BY *` 會透過所有的 `tag_key`  來聚合結果。

### time()

```sql
SELECT <function>(<field_key>) FROM_clause WHERE <time_range> GROUP BY time(<time_interval>[,<offset_interval>]),[tag_key] [fill(<fill_option>)]
```

- `time(time_interval)`：InfluxDB 根據 `time_interval` 對時間範圍進行分組，分組出來的結果包下不包上。如下範例：

    ```sql
    > SELECT * FROM "test" 
    name: test
    --------------
    time                   value
    2015-08-18T00:00:00Z   1
    2015-08-18T00:12:00Z   1
    2015-08-18T00:18:00Z   1
    
    > SELECT COUNT("value") FROM "test" GROUP BY time(12m)
    
    name: test
    --------------
    time                   count
    2015-08-18T00:00:00Z   1
    2015-08-18T00:12:00Z   2
    
    -- 2015-08-18T00:00:00Z： 2015-08-18T00:00:00Z <= time < 2015-08-18T00:12:00Z
    -- 2015-08-18T00:12:00Z： 2015-08-18T00:12:00Z <= time < 2015-08-18T00:24:00Z
    ```

  `time_interval` 的格式為 `uint` + `time units`， `time units` 可參考以下表格：

  ![](SELECT%20428e9b5f78da45a4ad17c6ebf1f8e067/Untitled.png)

- `time(time_interval,offset_interval)`： InfluxDB 根據 `offset_interval` 調整時間邊界，並根據 `time_interval` 對時間範圍進行分組。

  `offset_interval` 的格式為 `int(可以為負)` + `time units`。

  當未指定 `offset_interval` 時，InfluxDB會以預設的時間邊界開始分組。如下範例：

    ```sql
    > SELECT * FROM test
    name: test
    time                 host value
    ----                 ---- -----
    2020-11-01T00:00:00Z A    0
    2020-11-01T00:06:00Z A    1
    2020-11-01T00:12:00Z A    2
    2020-11-01T00:18:00Z A    3
    2020-11-01T00:24:00Z A    4
    2020-11-01T00:36:00Z A    5
    
    -- GROUP BY time(8m)，預設從 00:00:00 開始每8分鐘為一組，COUNT 在 WHERE 時間範圍內的筆數
    > SELECT COUNT("value") FROM "test" WHERE time >= '2020-11-01T00:06:00Z' AND time <= '2020-11-01T00:36:00Z' GROUP BY time(8m)
    name: test
    time                 count
    ----                 -----
    2020-11-01T00:00:00Z 1
    2020-11-01T00:08:00Z 1
    2020-11-01T00:16:00Z 1
    2020-11-01T00:24:00Z 1
    2020-11-01T00:32:00Z 1
    
    -- GROUP BY time(8m,6m)，從 00:06:00 開始每8分鐘為一組，COUNT 在 WHERE 時間範圍內的筆數
    > SELECT COUNT("value") FROM "test" WHERE time >= '2020-11-01T00:06:00Z' AND time <= '2020-11-01T00:36:00Z' GROUP BY time(8m,6m)
    name: test
    time                 count
    ----                 -----
    2020-11-01T00:06:00Z 2
    2020-11-01T00:14:00Z 1
    2020-11-01T00:22:00Z 1
    2020-11-01T00:30:00Z 1
    ```

- `fill(<fill_option>)`：選填，預設情況下當組別沒有資料時會返回 `NULL`，該選項用來改變當沒有資料時返回的結果。

  `fill_option` 有以下值－

    1. `null`：包含該組別但值返回 `NULL`，此為默認值。

        ```sql
        > SELECT MAX("value") FROM "cpu" GROUP BY time(1s)
        name: cpu
        time                max
        ----                ---
        1607937488000000000 1
        1607937489000000000
        1607937490000000000
        1607937491000000000 2
        1607937492000000000 3
        1607937493000000000 4
        ```

    2. `numerical value`：返回該數值。

        ```sql
        > SELECT MAX("value") FROM "cpu" GROUP BY time(1s) fill(100)
        name: cpu
        time                max
        ----                ---
        1607937488000000000 1
        1607937489000000000 100
        1607937490000000000 100
        1607937491000000000 2
        1607937492000000000 3
        1607937493000000000 4
        ```

    3. `linear`： 根據 [linear interpolation](https://zh.wikipedia.org/wiki/%E7%BA%BF%E6%80%A7%E6%8F%92%E5%80%BC) 顯示值。

        ```sql
        > SELECT MAX("value") FROM "cpu" GROUP BY time(1s) fill(linear)
        name: cpu
        time                max
        ----                ---
        1607937488000000000 1
        1607937489000000000 1.3333333333333333
        1607937490000000000 1.6666666666666665
        1607937491000000000 2
        1607937492000000000 3
        1607937493000000000 4
        ```

    4. `none`：不顯示該組別。

        ```sql
        > SELECT MAX("value") FROM "cpu" GROUP BY time(1s) fill(none)
        name: cpu
        time                max
        ----                ---
        1607937488000000000 1
        1607937491000000000 2
        1607937492000000000 3
        1607937493000000000 4
        ```

    5. `previous`：顯示前一個組別的結果。

        ```sql
        > SELECT MAX("value") FROM "cpu" GROUP BY time(1s) fill(previous)
        name: cpu
        time                max
        ----                ---
        1607937488000000000 1
        1607937489000000000 1
        1607937490000000000 1
        1607937491000000000 2
        1607937492000000000 3
        1607937493000000000 4
        ```


## ORDER BY time

將結果按照時間排序。

```sql
SELECT_clause [INTO_clause] FROM_clause [WHERE_clause] [GROUP_BY_clause] ORDER BY time ASC|DESC 
```

- 預設使用 `ASC` 排序
- 在 influxQL 中只能用 `time` 排序

## LIMIT and SLIMIT

### LIMIT  <N>

`LIMIT <N>` 返回前 `N` 個 `Points`。

```sql
SELECT_clause [INTO_clause] FROM_clause [WHERE_clause] [GROUP_BY_clause] [ORDER_BY_clause] LIMIT <N>
```

### SLIMIT <N>

`SLIMIT <N>` 返回前 `N` 個 `Series` 的所有 `Points`，注意必須要搭配 `GROUP BY` 才能使用。

```sql
SELECT_clause [INTO_clause] FROM_clause [WHERE_clause] GROUP BY *[,time(<time_interval>)] [ORDER_BY_clause] SLIMIT <N>
```

### LIMIT <N1> SLIMIT <N2>

返回 `N2` 個 `Series`，每個 `Series` 返回 `N1` 個 `Points`

```sql
SELECT_clause [INTO_clause] FROM_clause [WHERE_clause] GROUP BY *[,time(<time_interval>)] [ORDER_BY_clause] LIMIT <N1> SLIMIT <N2>
```

## OFFSET and SOFFSET

### OFFSET <N>

返回資料時，跳過前面 `N` 筆 `Points`

```sql
SELECT_clause [INTO_clause] FROM_clause [WHERE_clause] [GROUP_BY_clause] [ORDER_BY_clause] LIMIT_clause OFFSET <N> [SLIMIT_clause]
```

### SOFFSET <N>

返回資料時，跳過前面 `N` 個 `Series`，注意必須要搭配 `GROUP BY` 和 `SLIMIT` 才能使用。

```sql
SELECT_clause [INTO_clause] FROM_clause [WHERE_clause] GROUP BY *[,time(time_interval)] [ORDER_BY_clause] [LIMIT_clause] [OFFSET_clause] SLIMIT_clause SOFFSET <N>
```

## Time Zone tz()

預設情況下 InfluxDB 返回 UTC+0 時間，可以在 `SELECT` 語句的末尾加上 `tz('時區')` ，使其將 time 的時區進行轉換。

```sql
SELECT_clause [INTO_clause] FROM_clause [WHERE_clause] [GROUP_BY_clause] [ORDER_BY_clause] [LIMIT_clause] [OFFSET_clause] [SLIMIT_clause] [SOFFSET_clause] tz('<time_zone>')
```

- 時間格式必須採用 `RFC3339 format` 才會進行轉換。
- `<time_zone>` 可參考 [List of tz database time zones(WIKI)](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones#List) 。

## 參考

[Explore data using InfluxQL - Influxdata 文檔](https://docs.influxdata.com/influxdb/v1.8/query_language/explore-data/#the-basic-select-statement)