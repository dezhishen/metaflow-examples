package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/OpenListTeam/metaflow"
	_ "github.com/OpenListTeam/metaflow/http_withoutbuffer"
	"github.com/gin-gonic/gin"
)

const uploadDir = "./uploads"

func main() {
	go startGinServer()
	time.Sleep(3 * time.Second) // 等待服务器启动
	fmt.Println("服务器已启动，监听端口 8080")

	wMeta := &metaflow.StreamMetadata{
		URL: "http://localhost:8080/raw/test",
		Metadata: map[string]string{
			"http-method": "PUT",
		},
	}
	w, err := metaflow.CreateStream(wMeta)
	if err != nil {
		fmt.Println("创建 StreamWriter 失败:", err)
		return
	}
	w.Write([]byte("Hello, this is a test file content!"))
	fmt.Println("文件已上传")
	if err := w.Close(); err != nil {
		fmt.Println("关闭 StreamWriter 失败:", err)
		return
	}
	rMeta := &metaflow.StreamMetadata{
		URL: "http://localhost:8080/raw/test",
		Metadata: map[string]string{
			"http-method": "GET",
		},
	}

	r, err := metaflow.CreateStream(rMeta)
	if err != nil {
		fmt.Println("创建 StreamReader 失败:", err)
		return
	}
	content, err := io.ReadAll(r)
	if err != nil {
		fmt.Println("读取 StreamReader 失败:", err)
		return
	}
	fmt.Println("读取的文件内容:", string(content))
	if err := r.Close(); err != nil {
		fmt.Println("关闭 StreamReader 失败:", err)
		return
	}

}
func startGinServer() {
	// 确保上传目录存在
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, 0755)
	}

	r := gin.Default()

	// GET /raw/:file - 读取文件内容
	r.GET("/raw/:file", getFileHandler)

	// PUT /raw/:file - 上传/更新文件内容
	r.PUT("/raw/:file", putFileHandler)

	// 启动服务器
	if err := r.Run(":8080"); err != nil {
		fmt.Println("服务器启动失败:", err)
	}
}

// 获取文件内容
func getFileHandler(c *gin.Context) {
	// 获取文件名参数
	fileName := c.Param("file")
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件名不能为空"})
		return
	}

	// 构建完整文件路径（防止目录遍历攻击）
	filePath := filepath.Join(uploadDir, filepath.Clean(fileName))

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 读取文件内容
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}

	// 返回文件内容
	c.Data(http.StatusOK, "application/octet-stream", fileContent)
}

// 上传/更新文件内容
func putFileHandler(c *gin.Context) {
	// 获取文件名参数
	fileName := c.Param("file")
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件名不能为空"})
		return
	}

	// 构建完整文件路径（防止目录遍历攻击）
	filePath := filepath.Join(uploadDir, filepath.Clean(fileName))

	// 创建文件目录（如果不存在）
	fileDir := filepath.Dir(filePath)
	if _, err := os.Stat(fileDir); os.IsNotExist(err) {
		if err := os.MkdirAll(fileDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建目录失败"})
			return
		}
	}

	// 创建文件
	out, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建文件失败"})
		return
	}
	defer out.Close()

	// 读取请求体并写入文件
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取请求体失败"})
		return
	}

	if _, err := out.Write(body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入文件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "文件上传成功"})
}
