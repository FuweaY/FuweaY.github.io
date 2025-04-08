# Shared Key
gh-ost在每次遷移時會要求原表和新表共享相同的 `unique`且 `NOT NULL` 的key。

## 介紹

以下為原表：

```sql
CREATE TABLE tbl (
  id bigint unsigned not null auto_increment,
  data varchar(255),
  more_data int,
  PRIMARY KEY(id)
)
```

新增一個 `ts timestamp` 欄位後的新表：

```sql
CREATE TABLE tbl (
  id bigint unsigned not null auto_increment,
  data varchar(255),
  more_data int,
  ts timestamp,
  PRIMARY KEY(id)
)
```

在此案例中 `gh-ost` 除了會使用 `PRIMARY KEY` 從 `tb1` 複製資料到 gh-ost TABLE－ `_tbl_gho` 以外，同時會將 `binlog` 應用在 `tbl` 的事件也應用在 `_tbl_gho` 中。

為了應用 `binlog` 而必須要有 `shard key` 。例如： `UPDATE tb1` 會被轉換成 `REPLACE INTO _tbl_gho` ，由此來確保 `update` 。因此在此事例中， `gh-ost` 透過 `PRIMARY KEY(ID)` 來讓 `tb1` 和 `_tbl_gho` 是一對一的關係。

## 規則

原表和新表必須共相同的 `unique`且 `NOT NULL` 的KEY：

- 不必是 `PRIMARY KEY`
- KEY在原表和舊表只要有相同的欄位，允許有不同的名稱

在遷移開始時， `gh-ost` 會檢查原表和新表是否至少有一個共同的 `unique`且 `NOT NULL` 的KEY，如果找不到 `gh-ost` 會退出。如果有 `unique` 但沒有 `NOT NULL` 屬性的key，當確保沒有實際 `NULL` 的值時可以使用 `--allow-nullable-unique-key` 讓 `gh-ost` 運行。注意有實際 `NULL` 的值將破壞數據。

## 範例

```sql
create table some_table (
  id int not null auto_increment,
  ts timestamp,
  name varchar(128) not null,
  owner_id int not null,
  loc_id int not null,
  primary key(id),
  unique key name_uidx(name)
)
```

在此事例中有2個`unique`且 `NOT NULL` 的KEY：`PRIMARY KEY` 和 `name_uidx`

- 允許的ALTER操作：
    - `add column i int`
    - `add key owner_idx(owner_id)`
    - `add unique key owner_idx(owner_id, name)` - 注意：在此遷移期間不要寫入有衝突的行
    - `drop key name_uidx` - `PRIMARY KEY` 成為這兩張表的 `Shared Key`
    - `drop primary key, add primary key(owner_id, loc_id)` - `name_uidx` 成為這兩張表的 `Shared Key`
    - `change id bigint unsigned not null auto_increment` - `PRIMARY KEY` 修改資料型態，並未修改VALUE 是可以的操作
    - `drop primary key, drop key name_uidx, add primary key(name), add unique key id_uidx(id)` - 交換兩個KEY。但無論是 `id` 還是 `name` 都可以使用
- 不允許的ALTER操作：
    - `drop primary key, drop key name_uidx` - 新表沒有 `unique` 且 `NOT NULL` 的KEY
    - `drop primary key, drop key name_uidx, create primary key(name, owner_id)` - 原表和新表沒有 `Shared Key` ，即使新的 `primary key` 包含了 `name` ，但因為是複合主鍵，因此沒有辦法確保新表的唯一性
- 解決方案

  如果需要將`PRIMARY KEY` 和 `name_uidx` 改為使用其他欄位作為key，則需要進行2次遷移：

    1. `ADD UNIQUE KEY temp_pk (temp_pk_column,...)`
    2. `DROP PRIMARY KEY, DROP KEY temp_pk, ADD PRIMARY KEY (temp_pk_column,...)`