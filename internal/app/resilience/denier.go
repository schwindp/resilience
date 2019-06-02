/* SPDX-License-Identifier: MIT
 * Copyright © 2019-2020 Nadim Kobeissi <nadim@nadim.computer>.
 * All Rights Reserved. */
package main

import (
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/elazarl/goproxy"
	"github.com/sqweek/dialog"
	"golang.org/x/crypto/blake2b"
)

func denierHostsInit() error {
	err := denierVerifyConfig()
	if err != nil {
		denierHostsError()
		return err
	}
	hosts, err := denierHostsRead()
	if err != nil {
		return err
	}
	denierUpdate(hosts, false)
	return err
}

func denierProxyInit() {
	stateX.proxy = goproxy.NewProxyHttpServer()
	stateX.proxy.Verbose = false
	stateX.proxy.OnRequest().HandleConnectFunc(denierProxyHandler)
	http.ListenAndServe(":7341", stateX.proxy)
}

func denierProxyHandler(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
	if !stateX.enabled {
		return goproxy.OkConnect, host
	}
	if adblockShouldBlock(stateX.rules, ctx.Req.URL.String(), map[string]interface{}{
		"domain": host,
	}) {
		return goproxy.RejectConnect, host
	}
	return goproxy.OkConnect, host
}

func denierUpdate(hosts []byte, write bool) error {
	var newRules []string
	var err error
	for _, rule := range strings.Split(string(hosts), "\n") {
		rule = strings.Trim(rule, "\r\n ")
		if len(rule) > 0 {
			newRules = append(newRules, rule)
		}
	}
	tempRules, err := adblockNewRules(newRules)
	if err != nil {
		denierUpdateError()
		return err
	}
	stateX.rules = tempRules
	newHash := blake2b.Sum256(hosts)
	stateX.hostsHash = strings.Join([]string{
		hex.EncodeToString(newHash[:]),
		"blockList",
	}, "  ")
	tempRules = nil
	if write {
		err = denierVerifyConfig()
		if err != nil {
			denierHostsError()
			return err
		}
		err = denierHostsWrite(hosts)
		if err != nil {
			denierHostsError()
			return err
		}
	}
	return err
}

func denierVerifyConfig() error {
	var err error
	currentUser, _ := user.Current()
	hostsFilePath := path.Join(path.Join(path.Join(
		currentUser.HomeDir, ".config"), "resilience"), "blockList",
	)
	configFolderInfo, err := os.Stat(
		path.Join(currentUser.HomeDir, ".config"),
	)
	if err != nil || !configFolderInfo.IsDir() {
		err = os.Mkdir(path.Join(currentUser.HomeDir, ".config"), 0700)
		if err != nil {
			return err
		}
	}
	configFolderInfo, err = os.Stat(path.Join(
		currentUser.HomeDir,
		path.Join(".config", "resilience"),
	))
	if err != nil || !configFolderInfo.IsDir() {
		err = os.Mkdir(path.Join(
			currentUser.HomeDir,
			path.Join(".config", "resilience")),
			0700,
		)
		if err != nil {
			return err
		}
	}
	_, err = os.OpenFile(hostsFilePath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	return err
}

func denierHostsRead() ([]byte, error) {
	currentUser, _ := user.Current()
	hostsFilePath := path.Join(path.Join(path.Join(
		currentUser.HomeDir, ".config"), "resilience"), "blockList",
	)
	hosts, err := ioutil.ReadFile(hostsFilePath)
	return hosts, err
}

func denierHostsWrite(hosts []byte) error {
	currentUser, _ := user.Current()
	hostsFilePath := path.Join(path.Join(path.Join(
		currentUser.HomeDir, ".config"), "resilience"), "blockList",
	)
	err := ioutil.WriteFile(hostsFilePath, hosts, 0600)
	return err
}

func denierUpdateError() {
	dialog.Message(stateX.locale.denierUpdateErrorText).Title(stateX.locale.errorTitle).Error()
}

func denierHostsError() {
	dialog.Message(stateX.locale.denierHostsErrorText).Title(stateX.locale.errorTitle).Error()
}
