package main

import (
	"fmt"
	"net"
	"strings"
)

type User struct {
	Addr    string
	Name    string
	Conn    net.Conn
	Server  *Server
	Message chan string
}

func NewUser(conn net.Conn, server *Server) *User {
	addr := conn.RemoteAddr().String() //conn.RemoteAddr().String() 用于获取当前网络连接的远程地址（客户端的地址）
	user := &User{
		Addr:    addr,
		Name:    addr,
		Conn:    conn,
		Server:  server,
		Message: make(chan string, 1024), // 增加缓冲区
	}
	return user
}

// 封装发送消息给client的函数
func (this *User) SendToClient(msg string) {

	_, err := this.Conn.Write([]byte(msg))
	if err != nil {
		fmt.Printf("User.SendToClient发送消息失败:%s", err)
	} else {
		fmt.Printf("User.SendToClient消息已发送:%s", msg)
	}
}

// Message管道中一有消息就发给client
func (this *User) Listen() {
	for msg := range this.Message {
		// fmt.Printf("User.Listen向客户端发送消息:%s", msg) // 调试信息
		this.SendToClient(msg)
	}
}

// 处理消息的逻辑
func (this *User) DoMessage(msg string) {
	switch {
	case strings.HasPrefix(msg, "ModifyName:"):
		this.handleModifyName(msg[len("ModifyName:"):])
	case strings.HasPrefix(msg, "PublicChat:"):
		this.handlePublicChat(msg[len("PublicChat:"):])
	case strings.HasPrefix(msg, "SelectUser"):
		this.handleListUsers()
	case strings.HasPrefix(msg, "To|"):
		this.handlePrivateChat(msg)
	default:
		this.SendToClient("未知命令\n")
	}
}

// 修改用户名
func (this *User) handleModifyName(newName string) {
	newName = strings.TrimSpace(newName)
	if len(newName) == 0 {
		this.SendToClient("错误：用户名不能为空\n")
		return
	}

	this.Server.UserLock.Lock()
	defer this.Server.UserLock.Unlock()

	if _, exists := this.Server.NameToAddr[newName]; exists {
		this.SendToClient("错误：用户名已存在\n")
		return
	}

	// 更新映射
	delete(this.Server.NameToAddr, this.Name)
	this.Server.NameToAddr[newName] = this.Addr
	this.Name = newName

	this.SendToClient(fmt.Sprintf("用户名已修改为: %s\n", newName))
}

// 私聊
func (this *User) handlePrivateChat(msg string) {
	parts := strings.SplitN(msg, "|", 3)
	if len(parts) < 3 {
		this.SendToClient("错误：私聊格式错误\n")
		return
	}

	targetName := strings.TrimSpace(parts[1])
	content := parts[2]

	this.Server.UserLock.RLock()
	defer this.Server.UserLock.RUnlock()

	targetAddr, exists := this.Server.NameToAddr[targetName]
	if !exists {
		this.SendToClient("错误：用户不存在\n")
		return
	}

	targetUser := this.Server.UserMap[targetAddr]
	targetUser.SendToClient(fmt.Sprintf("[私聊]%s: %s\n", this.Name, content))
	this.SendToClient(fmt.Sprintf("发送给 %s: %s\n", targetName, content))
}

// 公聊
func (this *User) handlePublicChat(content string) {
	msg := fmt.Sprintf("[公聊]%s: %s\n", this.Name, content)
	this.Server.SendToAll(msg)
}

// 获取在线用户列表
func (this *User) handleListUsers() {
	this.Server.UserLock.RLock()
	defer this.Server.UserLock.RUnlock()

	var users []string
	for name := range this.Server.NameToAddr {
		users = append(users, name)
	}
	this.SendToClient("在线用户:\n" + strings.Join(users, "\n") + "\n")
}

// 用户上线
func (this *User) Online() {
	go this.Listen()

	msg := fmt.Sprintf("User.Online[%s]:%s上线了\n", this.Addr, this.Name)

	this.Server.UserLock.Lock()
	this.Server.UserMap[this.Addr] = this
	this.Server.NameToAddr[this.Name] = this.Addr
	this.Server.UserLock.Unlock()

	this.Server.SendToAll(msg)

}

// 用户下线
func (this *User) Offline() {
	msg := fmt.Sprintf("User.Offline[%s]:%s下线了\n", this.Addr, this.Name)

	this.Server.UserLock.Lock()
	delete(this.Server.UserMap, this.Addr)
	delete(this.Server.NameToAddr, this.Name)
	this.Server.UserLock.Unlock()

	this.Server.SendToAll(msg)
	close(this.Message)
	this.Conn.Close()
}
