---
title: 全文檢索
description: 介紹全文檢索
slug: fulltext-search
date: 2024-11-20T12:00:00+08:00
categories:
   - ElasticSearch
   - MySQL
weight: 1  
---
## 全文檢索
假設今天在 MySQL 下有一張叫 article 的表，我們想要找出 content(內文) 中包含 `金錢` 兩個字的所有文章，就需要使用以下模糊查詢：

```sql
SELECT * FROM article WHERE content LIKE '%莎士比亞%';
```

在這種情況下 MySQL 需要全表掃描整張表的 content 才能找出結果，因此搜尋效率非常的不好。

這個時候我們就去要透過建立 FullText Index 或者是使用搜尋引擎(solr、elasticsearch...)來優化查詢。

## 精確匹配和相關性匹配
還有什麼原因需要搜尋引擎呢？

假設我們有一下資料：

| id | content |
| --- | --- |
| 1 | 莎士比亞的故事 |
| 2 | 哈姆雷特作者的故事 |
| 3 | 羅密歐與茱麗葉作者的故事 |

當使用者輸入 `沙士比壓`，如果我們使用 MySQL 進行一般的模糊查詢：

```sql
SELECT * FROM article WHERE content LIKE '%沙士比壓%';
```

很顯然這樣我們不會得到任何結果，因為這是一個要求「**精確匹配**」的「模糊查詢」。

```sql
GET /article/_doc/_search?q=content:沙士比壓
```

在使用資料庫搜索時，我們更多的是基於「精確匹配」的搜索。

什麼是「精確匹配」？

比如搜訂單，根據訂單狀態，準確搜索。 搜「已完成」，就要「精確匹配」「已完成」的訂單，搜「待支付」，就要「精確匹配」「待支付」的訂單。

這種「精確匹配」的搜索能力，傳統關係型資料庫是非常勝任的。

**和「精確匹配」相比，「相關性匹配」更貼近人的思維方式。**

比如我要搜一門講過「莎士比亞」的課程，我需要在課程的文稿里進行「相關性匹配」，找到對應的文稿，你可能覺得一條 sql 語句就可以解決這個問題：

```sql
select * from course where content like "%莎士比亚%"
```

然而，這隻能算是「模糊查詢」，用你要搜索的字串，去「精確」的「模糊查詢」，其實還是「精確匹配」，機械思維。

那麼到底什麼是「相關性匹配」，什麼才是「人的思維」呢？

比如我搜「莎士比亞」，我要的肯定不只是精精確確包含「莎士比亞」的文稿，我可能還要搜「莎翁」、「Shakespeare」、「哈姆雷特」、「羅密歐和朱麗葉」、「威尼斯的商人」...

又比如我輸錯了，輸成「莎士筆亞」，「相關性匹配」可以智慧的幫我優化為「莎士比亞」，返回對應的搜尋結果。

這就是搜尋引擎的強大之處，它似乎可以理解你的真實意圖。

## 原理

[Inverted index 反向索引]({{< ref "post/ElasticSearch/inverted-index/index.md" >}})

[analyzer 分詞器]({{< ref "post/ElasticSearch/analyzer/index.md" >}})

## 工具

[FullText Index(全文檢索)](%E5%85%A8%E6%96%87%E6%AA%A2%E7%B4%A2%20cc4f9acfadd04425881d7d39fcc1f64f/FullText%20Index(%E5%85%A8%E6%96%87%E6%AA%A2%E7%B4%A2)%200af8dd34c3fc44c5802274304eb15c0d.md)

[elastic](%E5%85%A8%E6%96%87%E6%AA%A2%E7%B4%A2%20cc4f9acfadd04425881d7d39fcc1f64f/elastic%2074bc05f3e9d446de960cf24abaed72fa.md)

[Solr](%E5%85%A8%E6%96%87%E6%AA%A2%E7%B4%A2%20cc4f9acfadd04425881d7d39fcc1f64f/Solr%20cbe24c26bc804338972fed7067050bea.md)

## 參考

[为什么需要 Elasticsearch - 知乎 (zhihu.com)](https://zhuanlan.zhihu.com/p/73585202)