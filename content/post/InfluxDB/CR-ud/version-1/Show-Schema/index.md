---
title: InfluxDB 1.X show schema
description: 介紹 InfluxDB 1.X show schema
slug: influxdb-1-CR-ud/show-schema
date: 2021-01-22T12:00:00+08:00
categories:
   - InfluxDB
weight: 1  
---
# Show Schema

### SHOW DATABASE

```sql
SHOW <database_name>
```

### SHOW Retention policy

```sql
SHOW RETENTION POLICIES [ON <database_name>]
```

- ON <database_name> 雖然是選填的，但是若不帶入必須先運行 `USE <database_name>`。

### SHOW Measurement

```sql
SHOW MEASUREMENTS [ON <database_name>] [WITH MEASUREMENT <regular_expression>] [WHERE <tag_key> <operator> ['<tag_value>' | <regular_expression>]] [LIMIT_clause] [OFFSET_clause]
```

- ON <database_name> 雖然是選填的，但是若不帶入必須先運行 `USE <database_name>`。

### SHOW Series

```sql
SHOW SERIES [ON <database_name>] [FROM_clause] [WHERE <tag_key> <operator> [ '<tag_value>' | <regular_expression>]] [LIMIT_clause] [OFFSET_clause]
```

- ON <database_name> 雖然是選填的，但是若不帶入必須先運行 `USE <database_name>`。
- `WHERE` 條件不支持 `field` 判斷
- `WHERE` 條件中的 `operator` 支持 `等於 =`、`不等於 <> OR !=`、 `LIKE =~`、 `NOT LIKE !~`
- `WHERE` 條件中還能帶入 `time` 限縮，只有當條件為 `time` 時才可以使用 `>` `<` 。
  當以時間為條件時，實際上是使用 `shard` 為粒度，會判斷該時間所屬哪一個 `shard` 以此進行篩選，所以會出現超出 `time` 範圍的資料，如下範例：

    ```sql
    > show shards
    name: hour_test
    id database  retention_policy shard_group start_time           end_time             expiry_time          owners
    -- --------  ---------------- ----------- ----------           --------             -----------          ------
    50 hour_test mytest           50          2020-12-08T03:00:00Z 2020-12-08T04:00:00Z 2020-12-15T04:00:00Z
    52 hour_test mytest           52          2020-12-08T04:00:00Z 2020-12-08T05:00:00Z 2020-12-15T05:00:00Z
    51 hour_test mytest           51          2020-12-08T05:00:00Z 2020-12-08T06:00:00Z 2020-12-15T06:00:00Z
    53 hour_test mytest           53          2020-12-08T06:00:00Z 2020-12-08T07:00:00Z 2020-12-15T07:00:00Z
    54 hour_test mytest           54          2020-12-08T07:00:00Z 2020-12-08T08:00:00Z 2020-12-15T08:00:00Z
    
    > select * from test
    name: test
    time                test_key_12 test_key_13 test_key_init value
    ----                ----------- ----------- ------------- -----
    1607399129788792227                         0             1
    1607400000000000000 12                                    1
    1607403600000000000             13                        1
    
    > show series  where time > 1607400000000000000
    key
    ---
    test,test_key_12=12
    test,test_key_13=13
    ```


### SHOW Tag Keys

```sql
SHOW TAG KEYS [ON <database_name>] [FROM_clause] [WHERE <tag_key> <operator> ['<tag_value>' | <regular_expression>]] [LIMIT_clause] [OFFSET_clause]
```

- ON <database_name> 雖然是選填的，但是若不帶入必須先運行 `USE <database_name>`。
- `WHERE` 條件不支持 `field` 判斷
- `WHERE` 條件中的 `operator` 支持 `等於 =`、`不等於 <> OR !=`、 `LIKE =~`、 `NOT LIKE !~`
- `WHERE` 條件中還能帶入 `time` 限縮，只有當條件為 `time` 時才可以使用 `>` `<` 。
  當以時間為條件時，實際上是使用 `shard` 為粒度，會判斷該時間所屬哪一個 `shard` 以此進行篩選，所以會出現超出 `time` 範圍的資料。

### SHOW Tag Values

```sql
SHOW TAG VALUES [ON <database_name>][FROM_clause] WITH KEY [ [<operator> "<tag_key>" | <regular_expression>] | [IN ("<tag_key1>","<tag_key2")]] [WHERE <tag_key> <operator> ['<tag_value>' | <regular_expression>]] [LIMIT_clause] [OFFSET_clause]
```

- ON <database_name> 雖然是選填的，但是若不帶入必須先運行 `USE <database_name>`。
- `WHERE` 條件不支持 `field` 判斷
- `WITH` 和 `WHERE` 條件中 `operator` 支持 `等於 =`、`不等於 <> OR !=`、 `LIKE =~`、 `NOT LIKE !~`
- `WITH` 為必填支援指定的 `tag_key`、正則表達式和多個 `tag_key (IN)`

    ```sql
    > SHOW TAG VALUES ON "NOAA_water_database" WITH KEY IN ("location","randtag") WHERE "randtag" =~ /./ LIMIT 3
    
    name: h2o_quality
    key        value
    ---        -----
    location   coyote_creek
    location   santa_monica
    randtag	   1
    ```

- `WHERE` 條件中還能帶入 `time` 限縮，只有當條件為 `time` 時才可以使用 `>` `<` 。
  當以時間為條件時，實際上是使用 `shard` 為粒度，會判斷該時間所屬哪一個 `shard` 以此進行篩選，所以會出現超出 `time` 範圍的資料。

### SHOW Field Keys

```sql
SHOW FIELD KEYS [ON <database_name>] [FROM <measurement_name>]
```

- ON <database_name> 雖然是選填的，但是若不帶入必須先運行 `USE <database_name>`。
- `WHERE` 條件中還能帶入 `time` 限縮，只有當條件為 `time` 時才可以使用 `>` `<` 。
  當以時間為條件時，實際上是使用 `shard` 為粒度，會判斷該時間所屬哪一個 `shard` 以此進行篩選，所以會出現超出 `time` 範圍的資料。
- `Field Values` 在不同的 `shard` 允許不同的資料型態，因此 `SHOW FIELD KEYS` 會返回所有的資料型態，如下範例：

    ```sql
    > SHOW FIELD KEYS FROM cpu
    
    name: cpu
    fieldKey        fieldType
    --------        ---------
    all_the_types   integer
    all_the_types   float
    all_the_types   string
    all_the_types   boolean
    ```