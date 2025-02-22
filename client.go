package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type Client struct {
	ServerIp   string
	ServerPort int
	Conn       net.Conn
	Flag       int
	reader     *bufio.Reader
}

var serverPort int
var serverIp string

func NewClient(ip string, port int) *Client {
	client := &Client{
		ServerIp:   ip,
		ServerPort: port,
		reader:     bufio.NewReader(os.Stdin),
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", client.ServerIp, client.ServerPort))
	if err != nil {
		fmt.Println("dial error", err)
	}
	client.Conn = conn
	return client
}
func (client *Client) SendToServer(msg string) {
	if _, err := client.Conn.Write([]byte(msg + "\n")); err != nil {
		fmt.Println("发送失败:", err)
	}
}

// 处理server回应的消息， 直接显示到标准输出即可
func (client *Client) DealResponse() {
	fmt.Println("开始监听来自服务器的消息...")
	scanner := bufio.NewScanner(client.Conn) //创建一个 bufio.Scanner 对象，用于从服务器的连接 client.Conn 中逐行读取数据。
	for scanner.Scan() {                     //scanner.Scan() 会读取一行数据，直到遇到换行符 \n 或文件结束符（EOF）。
		fmt.Println(scanner.Text())
	}

	//当服务器关闭连接或发生错误时，scanner.Scan() 会返回 false，循环结束

}
func init() {
	flag.IntVar(&serverPort, "port", 8888, "server port")
	flag.StringVar(&serverIp, "ip", "127.0.0.1", "server ip")
}
func (client *Client) Menu() bool {
	var flag int
	fmt.Println("1:公聊模式")
	fmt.Println("2:私聊模式")
	fmt.Println("3:更改用户名")
	fmt.Println("0:退出")

	_, err := fmt.Scanln(&flag)
	if err != nil {
		fmt.Println("输入错误,请输入合法范围内的数字")
		bufio.NewReader(os.Stdin).ReadString('\n') // 清空缓冲区中的换行符
		return false
	}

	if flag >= 0 && flag <= 3 {
		client.Flag = flag
		return true
	} else {
		fmt.Println("请输入合法范围内的数字")
		return false
	}

}

func (client *Client) Run() {
	for {
		for client.Menu() {
			switch client.Flag {
			case 1:
				fmt.Println("选择公聊模式")
				client.PublicChat()
			case 2:
				fmt.Println("选择私聊模式")
				client.PrivateChat()
			case 3:
				fmt.Println("更改用户名")
				client.ModifyName()
			case 0:
				fmt.Println("退出")
				client.exit()
				return
			default:
				fmt.Println("请输入合法范围内的数字")
			}
		}
	}
}

func (client *Client) PublicChat() {
	fmt.Println("请输入聊天内容，exit退出")
	for {
		input, _ := client.reader.ReadString('\n') //使用 bufio.Reader 从标准输入读取一行数据，直到遇到换行符 \n。
		input = strings.TrimSpace(input)
		if input == "exit" {
			return
		}
		client.SendToServer("PublicChat:" + input)

	}
}

func (client *Client) PrivateChat() {
	client.SendToServer("SelectUser:")
	time.Sleep(time.Millisecond * 100) // 等待服务器返回用户列表

	fmt.Println("请输入聊天对象:")

	target, _ := client.reader.ReadString('\n')
	target = strings.TrimSpace(target)

	fmt.Println("请输入聊天内容,输入exit退出")
	for {
		input, _ := client.reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "exit" {
			return
		}
		client.SendToServer(fmt.Sprintf("To|%s|%s", target, input))
	}
}

func (client *Client) ModifyName() {
	fmt.Println("输入新用户名:")
	name, _ := client.reader.ReadString('\n')
	client.SendToServer("ModifyName:" + strings.TrimSpace(name))
	fmt.Println("用户名修改成功")

}
func (client *Client) exit() {
	if client.Conn != nil {
		err := client.Conn.Close()
		if err != nil {
			fmt.Println("关闭连接时出错:", err)
		} else {
			fmt.Println("客户端连接已关闭")
		}
	}
}
func main() {
	//解析port和ip
	flag.Parse()
	//创建客户端
	client := NewClient(serverIp, serverPort)
	if client == nil {
		fmt.Println("client is nil, 连接失败")
		return
	}
	//单独开启一个goroutine去处理server的回执消息
	go client.DealResponse()
	fmt.Println("==============>连接成功")
	client.Run()
}
