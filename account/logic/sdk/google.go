package sdk

import (
	"server/pkg/pb"
)

type Google struct {
}

// func verifyIdToken(idToken string) (*oauth2.Tokeninfo, error) {
//	//var httpClient = &http.Client{}
//	//oauth2Service, err := oauth2.New(httpClient)
//	oauth2Service, err := oauth2.NewService(context.Background(), option.WithHTTPClient(http.DefaultClient))
//	if err != nil {
//		logger.Errorf("%v", err)
//		return nil, err
//	}
//	tokenInfoCall := oauth2Service.Tokeninfo()
//	tokenInfoCall.IdToken(idToken)
//	//tokenInfoCall.AccessToken()
//	tokenInfo, err := tokenInfoCall.Do()
//	if err != nil {
//		return nil, err
//	}
//	return tokenInfo, nil
// }

func (t *Google) Login(req *pb.C2SLogin) error {
	// accInfo, err := verifyIdToken(req.Token)
	// if err != nil {
	//	return err
	// }
	// if req.Uid != accInfo.UserId {
	//	return errors.New("user id is not same")
	// }

	return nil
}

//
// const (
//	PackageName  = "" //包名
//	ClientId     = "" //客户端id
//	ClientSecret = "" //客户端秘钥
//	//RedirectUrl="" //创建api项目时的重定向地址
//	RefreshToken = ""
// )
//
// type TokenInfo struct {
//	AccessToken string `json:"access_token"`
//	ExpiresIn   int    `json:"expires_in"`
//	Scope       string `json:"scope"`
//	TokenType   string `json:"token_type"`
// }
//
// //https://oauth2.googleapis.com/token
// func getAccessToken() *TokenInfo {
//	resp, err := http.PostForm("https://accounts.google.com/o/oauth2/token", url.Values{"grant_type": {"refresh_token"},
//		"client_id": {ClientId}, "client_secret": {ClientSecret}, "refresh_token": {RefreshToken}})
//	defer resp.Body.Close()
//	body, err := ioutil.ReadAll(resp.Body)
//	if err != nil {
//		fmt.Println("PostToken body err:", err)
//	}
//	fmt.Println("token:", string(body))
//	info := new(TokenInfo)
//	error := json.Unmarshal(body, &info)
//	if err != nil {
//		fmt.Println("PostToken Unmarshal err:", error)
//	}
//	fmt.Println("token info:", info)
//	return info
// }
//
// type googleLoginRet struct {
//	// 这6个字段是所有idToken都包含的
//	Iss string //"iss": "https://accounts.google.com",  //token签发者，值为https://accounts.google.com或者accounts.google.com
//	Sub string //"sub": "110169484474386276334", //用户在该Google应用中的唯一标识，类似于微信的OpenID
//	Azp string //"azp": "1008719970978-hb24n2dstb40o45d4feuo2ukqmcc6381.apps.googleusercontent.com", //具体我也不知道，猜测与aud相同，都是应用的client_id
//	Aud string //"aud": "1008719970978-hb24n2dstb40o45d4feuo2ukqmcc6381.apps.googleusercontent.com", //client_id
//	Iat string //"iat": "1433978353", //签发时间
//	Exp string //"exp": "1433981953", //过期时间
// }
//
// //https://oauth2.googleapis.com/tokeninfo?id_token={idToken}
// func (t *Google) LoginUseHttp(req *pb.MsgLogin) error {
//	info := getAccessToken()
//	addr := "https://oauth2.googleapis.com/tokeninfo"
//	param := make(url.Values)
//	param.Add("id_token", req.Token)
//	param.Add("access_token", info.AccessToken)
//
//	b, err := share.HttpGet(addr, []byte(param.Encode()), false)
//	if err != nil {
//		logger.Warnf("google login check err:%v:%s", err, req.Uid)
//		return err
//	}
//
//	logger.Debugf("login sdk ret:%s", string(b))
//
//	ret := &googleLoginRet{}
//	err = json.Unmarshal(b, ret)
//	if err != nil {
//		return errors.New("facebook ret unmarshal err:" + err.Error())
//	}
//	if ret.Sub != req.Uid {
//		return errSDKCheckFailed
//	}
//	return nil
// }
