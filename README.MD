# GO-DBF

## 功能

- [ ] 新建并保存dbf文件
- [x] 从现有DBF文件中加载
- [x] 将model写入到文件
- [x] 单条查询
- [x] 批量查询
- [x] 获取记录数
- [x] 重新加载
- [ ] 支持软删除
- [ ] 缓存并通过索引构建数据加速查询、过滤

## 安装

```bash
go get 
```

## Demo

### 初始化

```go
// 先定义一个与dbf列对应的结构体
type DBFRecord struct {
OrderType  string  `dbf:"order_type"`
PriceType  string  `dbf:"price_type"`
ModePrice  float32 `dbf:"mode_price"`
StockCode  string  `dbf:"stock_code"`
Volume     int     `dbf:"volume"`
AccountID  string  `dbf:"account_id"`
ActType    string  `dbf:"act_type"`
BrokerType string  `dbf:"brokertype"`
Strategy   string  `dbf:"strategy"`
Note       string  `dbf:"note"`
TradeParam string  `dbf:"tradeparam"`
InsertTime string  `dbf:"inserttime"`
BasketPath string  `dbf:"basketpath"`
}


// 初始化
func main() {
    fileName := "./data/XT_DBF_ORDER.dbf"
    dbf, err := NewDBFFromFile(fileName, "utf-8", &DBFRecord{})
}
```

### 新增一条数据

```go
func main() {
    fileName := "./data/XT_DBF_ORDER.dbf"
    dbf, err := NewDBFFromFile(fileName, "utf-8", &DBFRecord{})

    if err != nil {
        t.Fatalf("init failed %v", err)
    }
    r := DBFRecord{
        OrderType:  "23",
        PriceType:  "3",
        ModePrice:  2.35,
        StockCode:  "000001",
        Volume:     100,
        AccountID:  "000001",
        ActType:    "49",
        BrokerType: "2",
    }
    err = dbf.Append(&r)
}
```

### 查询单条数据

```go
func main() {
    fileName := "./data/XT_DBF_ORDER.dbf"
    dbf, err := NewDBFFromFile(fileName, "utf-8", &DBFRecord{})

    if err != nil {
        t.Fatalf("init failed %v", err)
    }
    r := DBFRecord{}
    // 读取第一条数据
    err = dbf.GetRecord(0, &r)
    if err != nil {
    	t.Fatalf("read failed %v", err)
    }
    fmt.Printf("DBF Record: %+v\n", r)
}
```

### 查询多条数据
```go
func main() {
    fileName := "./data/XT_DBF_ORDER.dbf"
    dbf, err := NewDBFFromFile(fileName, "utf-8", &DBFRecord{})

    if err != nil {
        t.Fatalf("init failed %v", err)
    }
    // 从474开始读到末尾
    rs := make([]DBFRecord, dbf.NumRecords()-474)
    err = dbf.GetRecords(474, dbf.header.NumRecords, &rs, 1)
    if err != nil {
        t.Fatalf("read failed %v", err)
    }
}
```