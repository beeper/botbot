package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/synapseadmin"
	"maunium.net/go/mautrix/util"
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
	if cfg.BeeperAPIURL != "" {
		resp, err := cli.Client.Get(cfg.BeeperAPIURL + "/check-username/" + url.PathEscape(username))
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
		var err error
		if useSynapseAPI {
			_, err = synadm.UsernameAvailable(ctx, username)
		} else {
			_, err = cli.RegisterAvailable(username)
		}
		if errors.Is(err, mautrix.MUserInUse) || errors.Is(err, mautrix.MExclusive) {
			return false, nil
		} else if err != nil {
			return false, err
		} else {
			return true, nil
		}
	}
}

func Login(ctx context.Context, userID id.UserID, password string) (*mautrix.RespLogin, error) {
	loginClient, _ := mautrix.NewClient(cfg.HomeserverURL, "", "")
	identifier := mautrix.UserIdentifier{
		Type: mautrix.IdentifierTypeUser,
		User: userID.String(),
	}
	const deviceDisplayName = "botbot"
	if cfg.LoginJWTKey != "" {
		return loginClient.Login(&mautrix.ReqLogin{
			Type:       mautrix.AuthTypeSynapseJWT,
			Identifier: identifier,
			Token:      createLoginToken(userID),

			InitialDeviceDisplayName: deviceDisplayName,
		})
	} else {
		return loginClient.Login(&mautrix.ReqLogin{
			Type:       mautrix.AuthTypePassword,
			Identifier: identifier,
			Password:   password,

			InitialDeviceDisplayName: deviceDisplayName,
		})
	}
}

func RegisterUser(ctx context.Context, username string) (string, error) {
	password := util.RandomString(72)
	if cfg.BeeperAPIURL != "" {
		return password, registerUserBeeper(ctx, username, password)
	} else if cfg.RegisterSecret != "" {
		return password, registerUserSynapse(ctx, username, password)
	} else {
		return "", fmt.Errorf("no way to register users configured")
	}
}

func registerUserSynapse(ctx context.Context, username, password string) error {
	_, err := synadm.SharedSecretRegister(ctx, cfg.RegisterSecret, synapseadmin.ReqSharedSecretRegister{
		Username:     username,
		Password:     password,
		UserType:     "bot",
		Admin:        false,
		InhibitLogin: true,
	})
	return err
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
	resp, err := cli.Client.Post(cfg.BeeperAPIURL+"/admin/bot/"+url.PathEscape(username), "application/json", &body)
	if err != nil {
		return fmt.Errorf("failed to send request to api server: %w", err)
	}
	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		return nil
	}
	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("failed to register: %s", respBody)
}
