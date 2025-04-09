---
title: InfluxDB 1.X CR-ud
description: 介紹 InfluxDB 1.X SQL LIKE CR-ud
slug: influxdb-1-CR-ud
date: 2021-01-22T12:00:00+08:00
categories:
   - InfluxDB
weight: 1  
---
# InfluxDB is not CRUD

首先我們要了解到 InfluxDB 是針對時間序列數據進行優化的數據庫，並且時間序列的數據通常只會寫入一次，很少會發生更新的情境，因此 InfluxDB 沒有完整的 `CRUD`，官方將其稱為 `CR-ud`。

相比 `UPDATE` 和 `DELETE` 更著重在 `CREATE` 和 `READ` 上，為了讓 `CREATE` 和 `READ` 性能更高， `UPDATE` 和 `DELETE` 操作有以下的限制：

1. 如果要 `UPDATE` 一個 `point`，只能透過 `INSERT` 一個具有相同 `series` + `timestamp` 的 `point`
2. 不可根據 `field value` 刪除數據，可以先透過 `READ` 取得 `timestamp`，隨後在透過 `timestamp` 進行刪除。
3. 不能夠 `UPDATE` 或 `RENAME` `tags` ，有關更多訊息可以參考 github
4.  不能夠透過 `tag key` 刪除 `tag`

---

## InfluxQL (v1.x SQL LIKE)

### [Database Manage Query]({{< ref "post/InfluxDB/CR-ud/version-1/Database-Mange-Query/index.md" >}})

### [Show Schema]({{< ref "post/InfluxDB/CR-ud/version-1/Show-Schema/index.md" >}})

### [INSERT]({{< ref "post/InfluxDB/CR-ud/version-1/Insert/index.md" >}})

### [SELECT]({{< ref "post/InfluxDB/CR-ud/version-1/Select/index.md" >}})

### [Continuous Queries]({{< ref "post/InfluxDB/CR-ud/version-1/continuous-queries/index.md" >}})

### [權限]({{< ref "post/InfluxDB/CR-ud/version-1/privilege/index.md" >}})

---

## FLUX (v2.0 JavaScript LIKE)

## 參考

[Influx Query Language (InfluxQL) reference -  influxdata 文檔](https://docs.influxdata.com/influxdb/v1.8/query_language/spec/)

[Explore your schema using InfluxQL - influxdata 文檔](https://docs.influxdata.com/influxdb/v1.8/query_language/explore-schema/)

[Manage your database - influxdata 文檔](https://docs.influxdata.com/influxdb/v1.8/query_language/manage-database/)

[Explore data using InfluxQL](https://docs.influxdata.com/influxdb/v1.8/query_language/explore-data/)