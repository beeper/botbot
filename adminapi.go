package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
)

type BeeperCheckUsernameResponse struct {
	Available bool   `json:"available"`
	Error     string `json:"error"`
}

const useSynapseAPI = true

var validUsernameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,28}bot$`)

func IsValidBotUsername(username string) bool {
	return validUsernameRegex.MatchString(username)
}

func IsUsernameAvailable(ctx context.Context, username string) (bool, error) {
	if cfg.BeeperAPI != "" {
		resp, err := cli.Client.Get(cfg.BeeperAPI + "/check-username/" + url.PathEscape(username))
		if err != nil {
			return false, fmt.Errorf("failed to send request to api server: %w", err)
		}
		var respData BeeperCheckUsernameResponse
		err = json.NewDecoder(resp.Body).Decode(&respData)
		if err != nil {
			return false, fmt.Errorf("failed to decode response from api server: %w", err)
		}
		return respData.Available, nil
	} else {
		var resp mautrix.RespRegisterAvailable
		var reqPath mautrix.PrefixableURLPath
		if useSynapseAPI {
			reqPath = mautrix.SynapseAdminURLPath{"v1", "username_available"}
		} else {
			reqPath = mautrix.ClientURLPath{"v3", "register", "available"}
		}
		reqURL := cli.BuildURLWithQuery(reqPath, map[string]string{"username": username})
		_, err := cli.MakeFullRequest(mautrix.FullRequest{
			Method:       http.MethodGet,
			URL:          reqURL,
			ResponseJSON: &resp,
			Context:      ctx,
		})
		if errors.Is(err, mautrix.MUserInUse) {
			return false, nil
		} else if err != nil {
			return false, fmt.Errorf("failed to send request to synapse: %w", err)
		}
		return resp.Available, nil
	}
}

type reqResetPassword struct {
	NewPassword   string `json:"new_password"`
	LogoutDevices bool   `json:"logout_devices"`
}

func ResetPassword(ctx context.Context, userID id.UserID, password string) error {
	req := &reqResetPassword{
		NewPassword:   password,
		LogoutDevices: true,
	}
	reqURL := cli.BuildURL(mautrix.SynapseAdminURLPath{"v1", "reset_password", userID})
	_, err := cli.MakeFullRequest(mautrix.FullRequest{
		Method:      http.MethodGet,
		URL:         reqURL,
		RequestJSON: &req,
		Context:     ctx,
	})
	return err
}

func Login(ctx context.Context, username, password string) (*mautrix.RespLogin, error) {
	loginClient, _ := mautrix.NewClient(cfg.HomeserverURL, "", "")
	resp, err := loginClient.Login(&mautrix.ReqLogin{
		Type: mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{
			Type: mautrix.IdentifierTypeUser,
			User: username,
		},
		Password:                 password,
		InitialDeviceDisplayName: "botbot",
	})
	return resp, err
}

func RegisterUser(ctx context.Context, username, password string) error {
	if cfg.BeeperAPI != "" {
		return registerUserBeeper(ctx, username, password)
	} else if cfg.RegisterSecret != "" {
		return registerUserSynapse(ctx, username, password)
	} else {
		return fmt.Errorf("no way to register users configured")
	}
}

type respGetSynapseAdminRegister struct {
	Nonce string `json:"nonce"`
}

type reqPostSynapseAdminRegister struct {
	Nonce        string `json:"nonce"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Displayname  string `json:"displayname,omitempty"`
	UserType     string `json:"user_type,omitempty"`
	InhibitLogin bool   `json:"inhibit_login,omitempty"`
	Admin        bool   `json:"admin"`
	Checksum     string `json:"mac"`
}

type respPostSynapseAdminRegister struct {
	AccessToken string `json:"access_token"`
	HomeServer  string `json:"home_server"`
	UserID      string `json:"user_id"`
	DeviceID    string `json:"device_id"`
}

func registerUserSynapse(ctx context.Context, username, password string) error {
	registerURL := cli.BuildURL(mautrix.SynapseAdminURLPath{"v1", "register"})
	var getRegister respGetSynapseAdminRegister
	_, err := cli.MakeFullRequest(mautrix.FullRequest{
		Method:       http.MethodGet,
		URL:          registerURL,
		ResponseJSON: &getRegister,
		Context:      ctx,
	})
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}
	reqRegister := reqPostSynapseAdminRegister{
		Nonce:        getRegister.Nonce,
		Username:     username,
		Password:     password,
		UserType:     "bot",
		InhibitLogin: true,
	}
	signer := hmac.New(sha1.New, []byte(cfg.RegisterSecret))
	signer.Write([]byte(reqRegister.Nonce))
	signer.Write([]byte{0})
	signer.Write([]byte(reqRegister.Username))
	signer.Write([]byte{0})
	signer.Write([]byte(reqRegister.Password))
	signer.Write([]byte{0})
	signer.Write([]byte("notadmin"))
	signer.Write([]byte{0})
	signer.Write([]byte("bot"))
	reqRegister.Checksum = hex.EncodeToString(signer.Sum(nil))
	var respRegister respPostSynapseAdminRegister
	_, err = cli.MakeFullRequest(mautrix.FullRequest{
		Method:       http.MethodPost,
		URL:          registerURL,
		RequestJSON:  &reqRegister,
		ResponseJSON: &respRegister,
		Context:      ctx,
	})
	if err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	}
	return nil
}

type reqBeeperRegister struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func registerUserBeeper(ctx context.Context, username, password string) error {
	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(&reqBeeperRegister{
		Username: username,
		Password: password,
	})
	if err != nil {
		return fmt.Errorf("failed to encode request body: %w", err)
	}
	resp, err := cli.Client.Post(cfg.BeeperAPI+"/admin/bot/"+url.PathEscape(username), "application/json", &body)
	if err != nil {
		return fmt.Errorf("failed to send request to api server: %w", err)
	}
	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		return nil
	}
	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("failed to register: %s", respBody)
}
