---
title: InfluxDB 1.X CQ
description: 介紹 InfluxDB 1.X Continuous Queries
slug: influxdb-1-CR-ud/continuous-queries
date: 2021-01-22T12:00:00+08:00
categories:
   - InfluxDB
weight: 1  
---
Continuous Queries(CQ) 類似於 MySQL 的 `Event` 可以自動定期的執行 query，並將結果儲存到特定的 measurement 。

## CQ 的使用場景

1. 採樣和數據保留：透過 CQ 和 RP 減輕儲存壓力，透過將高精度數據採樣成低精度數據，再讓 RP 將重要度不高的高精度數據 purge 掉。
2. 預先計算昂貴的查詢：透過使用 CQ 預先將高精度數據採樣到較低精度，再使用低精度的數據查詢將花費較少的時間。
3. 替代 HAVING 子句： InfluxDB 不支持 SQL 中的 `HAVING` 子句，因此可以透過 CQ 先將 GROUP BY 計算後的結果寫入另一個 `measurement`，隨後再透過 `WHERE` 條件從新的 `measurement` 中過濾資料。

## 基本 CQ 語法

```sql
CREATE CONTINUOUS QUERY <cq_name> ON <database_name>
BEGIN
  <cq_query>
END
```

## <cq_query>

```sql
SELECT <function[s]> 
INTO <destination_measurement> 
FROM <measurement> 
[WHERE <stuff>] 
GROUP BY time(<interval>)[,<tag_key[s]>]
```

1. `cq_query` 內中必須要有 `function`、 `INTO` 和 `GROUP BY time()` 這3個要素。
2. `GROUP BY time()` 中的 `<interval>`  同時也是 CQ 執行的頻率。
3. `cq_query` 中的 `WHERE` 不需要時間範圍，就算寫了也會被忽略，因為 `InfluxDB` 會在執行時自動帶入 `now()` ~ `now() - <interval>` 的時間範圍。
4. `GROUP BY time()` 除了 `interval` 也同樣可以加上 `offset_interval`。如下範例：

    ```sql
    CREATE CONTINUOUS QUERY "cq_cpu_average" ON "test"
    BEGIN
      SELECT mean("loading") INTO "cpu_average_loading" FROM "cpu" GROUP BY time(1h,15m)
    END
    ```

   假設在 `16:00` 新增此 CQ，則將在 `16:15` 執行 `15:15 ~ 16:14.99999999` 的資料運算。

5. 和一般的 `SELECT INTO` 一樣，可以在 INTO <destination_measurement> 中使用反向引用語法 `:MEASUREMEN`。在 FROM <measurement> 可以使用正則表達式。如下範例：

    ```sql
    CREATE CONTINUOUS QUERY "cq_average" ON "test"
    BEGIN
      SELECT mean(*) INTO "test"."autogen".:MEASUREMENT FROM /.*/ GROUP BY time(30m),*
    END
    ```

6. 基本 CQ 語法不支持使用 `fill()` 更改不含數據的間隔之回傳值，因此當該段區間沒有資料將不寫入任何結果，可以使用進階語法處理。
7. CQ 語法不會對舊有的區間執行，請手動使用 [`SELECT INTO`](SELECT%20428e9b5f78da45a4ad17c6ebf1f8e067.md) 語法。
8. 由於當語法中未包含 `GROUP BY *` 時，原表的 `tag` 在新表將被轉換為 `field`，因此原本用 `tag` 區別的數據被 `overwrite`，這可能導致數據的丟失。若要保留 `tag` 請務必加上 `GROUP BY *`。

## 進階 CQ 語法

```sql
CREATE CONTINUOUS QUERY <cq_name> ON <database_name>
RESAMPLE EVERY <interval> FOR <interval>
BEGIN
  <cq_query>
END
```

可以看到進階 CQ 語法多了 2 個元素－

### `EVERY <interval>`：定義 CQ 執行的間隔。

1. 假設 <interval> 在 `EVERY` 中小於(<) `GROUP BY`，則每 `EVERY` 時間執行查詢 `GROUP BY` 時間範圍的資料：

    ```sql
    CREATE CONTINUOUS QUERY "cq_every" ON "test"
    RESAMPLE EVERY 30m
    BEGIN
      SELECT mean("loading") 
        INTO "result_measurement"
        FROM "source_measurement"
        GROUP BY time(1h)
    END
    ```

   當沒有 `RESAMPLE EVERY 30m`：

    - 在 8:00 執行 CQ，資料範圍 [ 7:00 , 8:00 )
    - 在 9:00 執行 CQ，資料範圍 [ 8:00 , 9:00 )

   當有 `RESAMPLE EVERY 30m`：

    - 在 8:00 執行 CQ，資料範圍 [ 7:00 , 8:00 )
    - 在 8:30 執行 CQ，資料範圍 [ 8:00 , 9:00 )
    - 在 9:00 執行 CQ，資料範圍 [ 8:00 , 9:00 )，由於執行出來的 `time` 欄位和 8:30 相同，因此會覆蓋上一次 8:30 執行 CQ 的結果。
2. 假設 <interval> 在 `EVERY` 中等於(=) `GROUP BY`，則和基本語法相同沒有任何影響。
3. 假設 <interval> 在 `EVERY` 中大於(>) `GROUP BY`，則執行時間和時間範圍都以 `EVERY` 為主：

    ```sql
    CREATE CONTINUOUS QUERY "cq_every" ON "test"
    RESAMPLE EVERY 2h
    BEGIN
      SELECT mean("loading") 
        INTO "result_measurement"
        FROM "source_measurement"
        GROUP BY time(1h)
    END
    ```

   當沒有 `RESAMPLE EVERY 2h`：

    - 在 8:00 執行 CQ，資料範圍 [ 7:00 , 8:00 )
    - 在 9:00 執行 CQ，資料範圍 [ 8:00 , 9:00 )

   當有 `RESAMPLE EVERY 2h`：

    - 在 8:00 執行 CQ，資料範圍 [ 6:00 , 8:00 )
    - 在 10:00 執行 CQ，資料範圍 [ 8:00 , 10:00 )

### `FOR <interval>`：定義 CQ 執行時查詢的時間範圍。

1. 假設 <interval> 在 `FOR` 中大於(>) `GROUP BY`，則每隔 `GROUP BY` 時間執行查詢 `FOR` 時間範圍的資料，但會以 `GROUP BY` 時間分組：

    ```sql
    CREATE CONTINUOUS QUERY "cq_every" ON "test"
    RESAMPLE FOR 1h
    BEGIN
      SELECT mean("loading") 
        INTO "result_measurement"
        FROM "source_measurement"
        GROUP BY time(30m)
    END
    ```

   當沒有 `RESAMPLE FOR 1h`：

    - 在 8:00 執行 CQ 資料範圍 [ 7:30 , 8:00 )
    - 在 8:30 執行 CQ 資料範圍 [ 8:00 , 8:30 )

   當有 `RESAMPLE FOR 1h`：

    - 在 8:00 執行 CQ 資料範圍 [ 7:00 , 8:00 )，產生 2 個 `Point` [ 7:00 , 7:30 ) 和 [ 7:30 , 8:00 )
    - 在 8:30 執行 CQ 資料範圍 [ 7:30 , 8:30 )，產生 2 個 `Point` [ 7:30 , 8:00 ) 和 [ 8:00 , 8:30 )
    - 在 9:00 執行 CQ 資料範圍 [ 8:00 , 9:00 )，產生 2 個 `Point` [ 8:00 , 8:30 ) 和 [ 8:30 , 9:00 )
2. 假設 <interval> 在 `FOR` 中等於(=) `GROUP BY`，則和基本語法相同沒有任何影響。
3. 假設 <interval> 在 `FOR` 中小於(<) `GROUP BY`，則不允許建立此 CQ。

## SHOW CQ

顯示所有的 CQ ，會依照 database 進行分組顯示。

```sql
SHOW CONTINUOUS QUERIES
```

## DROP CQ

從一個指定的 database 刪除 CQ。

```sql
DROP CONTINUOUS QUERY <cq_name> ON <database_name>
```

## 修改 CQ

CQ 建立後無法修改，只能 `DROP` 之後重新 `CREATE`。