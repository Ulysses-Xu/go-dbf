package dbf

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

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

func getFileBuffer(fileName string) []byte {
	file, err := os.Open(fileName)
	defer file.Close()
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil
	}

	fileInfo, _ := file.Stat()
	buffer := make([]byte, fileInfo.Size())
	file.Read(buffer)
	return buffer
}
func CompareBytes(a, b []byte) [][3]int {
	var differences [][3]int // 存储不同位置和值的数组，元素类型为 [3]int，分别存储位置、a的值、b的值

	// 找到两个切片中较短的长度
	minLength := len(a)
	if len(b) < minLength {
		minLength = len(b)
	}

	// 遍历较短的长度
	for i := 0; i < minLength; i++ {
		if a[i] != b[i] {
			// 如果字节不同，记录位置和值
			differences = append(differences, [3]int{i, int(a[i]), int(b[i])})
		}
	}

	// 如果两个切片长度不同，则记录较长切片剩余的部分
	if len(a) > minLength {
		for i := minLength; i < len(a); i++ {
			differences = append(differences, [3]int{i, int(a[i]), -1}) // -1 表示b切片中没有对应的字节
		}
	} else if len(b) > minLength {
		for i := minLength; i < len(b); i++ {
			differences = append(differences, [3]int{i, -1, int(b[i])}) // -1 表示a切片中没有对应的字节
		}
	}

	return differences
}

func TestDBF_Append(t *testing.T) {
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
	if err != nil {
		t.Fatalf("append failed %v", err)
	}
}

func TestDBF_Read(t *testing.T) {
	fileName := "./data/XT_DBF_ORDER.dbf"
	dbf, err := NewDBFFromFile(fileName, "utf-8", &DBFRecord{})
	if err != nil {
		t.Fatalf("init failed %v", err)
	}
	//r := DBFRecord{}
	//err = dbf.GetRecord(0, &r)
	//if err != nil {
	//	t.Fatalf("read failed %v", err)
	//}
	//fmt.Printf("DBFHandler Record: %+v\n", r)

	rs := make([]DBFRecord, dbf.NumRecords()-474)
	err = dbf.GetRecords(474, dbf.header.NumRecords, &rs, 1)
	if err != nil {
		t.Fatalf("read failed %v", err)
	}

}

func Test_testChan(t *testing.T) {
	c := make(chan int, 10)
	wg := sync.WaitGroup{}
	go func(c chan int, wg *sync.WaitGroup) {
		for i := range c {
			fmt.Println(i)
			wg.Done()
		}
	}(c, &wg)
	//wg.Add(1)
	//c <- 1
	//wg.Wait()
	defer close(c)
}
