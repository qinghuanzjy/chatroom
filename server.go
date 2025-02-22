package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

type Server struct {
	Ip         string
	Port       int
	UserMap    map[string]*User
	NameToAddr map[string]string // Name -> Addr
	UserLock   sync.RWMutex
	Message    chan string
}

func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:         ip,
		Port:       port,
		UserMap:    make(map[string]*User),
		NameToAddr: make(map[string]string),
		Message:    make(chan string),
	}
	return server
}

// 封装把消息发给所有人的函数
func (this *Server) SendToAll(msg string) {
	this.UserLock.Lock()

	for _, user := range this.UserMap {
		select {
		case user.Message <- msg:
			fmt.Println("Server.SendToAll消息已发送给:", user.Name)
		default:
			log.Printf("用户 %s 的消息队列已满", user.Name)
		}
	}
	this.UserLock.Unlock()

}

// 一有消息就发给所有user的chan(将想要发的消息处理好后可以用这个函数）
func (this *Server) Listen() {
	fmt.Println("Server.Listen服务器开始监听")
	for msg := range this.Message {
		fmt.Println("Server.Listen服务器收到消息:", msg)
		this.SendToAll(msg)
		fmt.Println(msg, "Server.Listen消息已发送给所有用户")
	}
}

// 监听客户端的输入
func (this *Server) Handle(user *User) {
	// 成功接收到连接后打印当前所有用户的上线信息
	// this.UserLock.Lock()
	// for _, user := range this.UserMap {
	// 	fmt.Println("Server.Handle用户", user.Name, "在线")
	// }
	// this.UserLock.Unlock()

	go func() {
		buff := make([]byte, 4096)
		for {
			n, err := user.Conn.Read(buff)
			if n == 0 {
				user.Offline()
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("Server.Handle Conn Read err:", err)
				return
			}
			// fmt.Println("Server.Handle读取的长度为", n)
			msg := string(buff[:n-1]) //去掉最后的\n
			// fmt.Println("Server.Handle读取到的消息为", msg)
			user.DoMessage(msg)
		}
	}()
}
func (this *Server) Run() {
	fmt.Println("Server.Run服务器正在运行")
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Println("Server.Run服务器启动失败", err)
	}
	defer listener.Close()
	//启动监听
	go this.Listen()
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Server.Run连接失败", err)
			continue
		}
		user := NewUser(conn, this)
		// fmt.Println("Server.Run用户conn.RemoteAddr().String()", conn.RemoteAddr().String())
		user.Online()
		fmt.Println("Server.Run连接建立成功", user.Name)
		go this.Handle(user)
	}

}
