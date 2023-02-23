package crypt

import (
	"bytes"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	E "github.com/hadi77ir/wsproxy/pkg/errors"
	"github.com/hadi77ir/wsproxy/pkg/utils"
	utls "github.com/refraction-networking/utls"
	"net/url"
	"strings"
)

var ErrProfileNotSupported = errors.New("profile not supported by uTLS library")

const (
	ParamSNI                            = "tls.sni"
	ParamNextProtos                     = "tls.alpn"
	ParamHelloId                        = "tls.profile"
	ParamCertificate                    = "tls.cert"
	ParamPrivateKey                     = "tls.key"
	ParamCertificatePin                 = "tls.pin"
	ParamInsecure                       = "tls.insecure"
	ParamClientCA                       = "tls.clientca"
	CertificatePinDigestMethodSeparator = ":"
	MultipleValuesSeparator             = ","
	MultiplePathsSeparator              = ":"
)

func LoadCertPoolFromParams(parameters url.Values, paramName string) (*x509.CertPool, int, error) {
	pool := x509.NewCertPool()
	certificates, err := LoadCertsFromParams(parameters, paramName)
	if err != nil {
		return nil, 0, err
	}
	for _, cert := range certificates {
		pool.AddCert(cert)
	}
	return pool, len(certificates), nil
}

func LoadCertsFromParams(parameters url.Values, paramName string) ([]*x509.Certificate, error) {
	paths, found := utils.GetParameter(parameters, paramName)
	pathsSplit := strings.Split(paths, MultiplePathsSeparator)
	if found && len(paths) > 0 {
		certs := []*x509.Certificate{}
		for _, path := range pathsSplit {
			contents, err := utils.ReadFile(path)
			if err != nil {
				return nil, err
			}
			newCerts, err := x509.ParseCertificates(contents)
			if err != nil {
				return nil, err
			}
			certs = append(certs, newCerts...)
		}
		return certs, nil
	}
	return nil, nil
}

func LoadX509PairBytesFromParams(parameters url.Values) (cert []byte, key []byte, err error) {
	keyPath, keyPathFound := utils.GetParameter(parameters, ParamPrivateKey)
	certPath, certPathFound := utils.GetParameter(parameters, ParamCertificate)
	if keyPathFound && len(keyPath) > 0 && !(certPathFound && len(certPath) > 0) {
		return nil, nil, E.ErrMissingPart(ParamCertificate)
	}
	if !(keyPathFound && len(keyPath) > 0) && certPathFound && len(certPath) > 0 {
		return nil, nil, E.ErrMissingPart(ParamPrivateKey)
	}
	if keyPathFound && len(keyPath) > 0 && certPathFound && len(certPath) > 0 {
		return LoadX509PairBytes(certPath, keyPath)
	}
	// there is no error. as there was nothing to be loaded.
	return nil, nil, nil
}

func LoadX509PairBytes(certPath, keyPath string) (cert []byte, key []byte, err error) {
	key, err = utils.ReadFile(keyPath)
	if err != nil {
		return nil, nil, err
	}
	cert, err = utils.ReadFile(certPath)
	if err != nil {
		return nil, nil, err
	}
	return
}
func LoadX509PairsBytesFromParams(parameters url.Values) (certs [][]byte, keys [][]byte, err error) {
	keyPath, keyPathFound := utils.GetParameter(parameters, ParamPrivateKey)
	certPath, certPathFound := utils.GetParameter(parameters, ParamCertificate)
	if !keyPathFound && !certPathFound {
		return nil, nil, nil
	}
	if keyPathFound && len(keyPath) > 0 && !(certPathFound && len(certPath) > 0) {
		return nil, nil, E.ErrMissingPart(ParamCertificate)
	}
	if !(keyPathFound && len(keyPath) > 0) && certPathFound && len(certPath) > 0 {
		return nil, nil, E.ErrMissingPart(ParamPrivateKey)
	}
	keyPaths := strings.Split(keyPath, MultiplePathsSeparator)
	certPaths := strings.Split(certPath, MultiplePathsSeparator)
	if len(keyPaths) > len(certPaths) {
		return nil, nil, E.ErrMissingPart(ParamCertificate)
	}
	if len(keyPaths) < len(certPaths) {
		return nil, nil, E.ErrMissingPart(ParamPrivateKey)
	}
	pairCount := len(keyPaths)
	certs = make([][]byte, pairCount)
	keys = make([][]byte, pairCount)
	for i := 0; i < len(keyPaths); i++ {
		keys[i], certs[i], err = LoadX509PairBytes(certPaths[i], keyPaths[i])
		if err != nil {
			return nil, nil, err
		}
	}
	return
}

func GetCertificatePinningAndInsecure(parameters url.Values) (vFunc PeerVerifierFunc, insecureBool bool, err error) {
	if insecure, found := utils.GetParameter(parameters, ParamInsecure); found {
		insecureBool, err = utils.ParseBool(insecure)
		if err != nil {
			return nil, false, err
		}
		vFunc = insecureVerifier
	}

	if pin, found := utils.GetParameter(parameters, ParamCertificatePin); found {
		verifyFunc, err := getPinVerificationFunc(pin)
		if err != nil {
			return nil, false, err
		}
		return verifyFunc, true, nil
	}
	return
}

type PeerVerifierFunc func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error

var certificateNotMatchingPinErr = errors.New("certificate fingerprint doesn't match with the pinned hash")

type CertificatePin struct {
	DigestFunc func([]byte) []byte
	Digest     []byte
}

func getPinVerificationFunc(pin string) (PeerVerifierFunc, error) {
	if pin != "" {
		pinsSplit := strings.Split(pin, MultipleValuesSeparator)
		pins := make([]CertificatePin, len(pinsSplit))
		for i, pin := range pinsSplit {
			pinSplit := strings.SplitN(pin, CertificatePinDigestMethodSeparator, 2)
			pinBytes, err := hex.DecodeString(pinSplit[1])
			if err != nil {
				return nil, err
			}
			pins[i] = CertificatePin{DigestFunc: GetDigestFunc(pinSplit[0]), Digest: pinBytes}
			if pins[i].DigestFunc == nil {
				return nil, E.ErrOpNotSupported
			}
		}
		return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			if l := len(rawCerts); l != 1 {
				return fmt.Errorf("got len(rawCerts) = %d, wanted 1", l)
			}
			cert, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return err
			}
			for _, pin := range pins {
				if bytes.Equal(pin.DigestFunc(cert.RawSubjectPublicKeyInfo), pin.Digest) {
					return nil
				}
			}
			return certificateNotMatchingPinErr
		}, nil
	}
	return nil, nil
}

func insecureVerifier(_ [][]byte, _ [][]*x509.Certificate) error {
	return nil
}

func GetSNIFromParams(parameters url.Values) string {
	return utils.StringFromParameters(parameters, ParamSNI, "")
}
func GetNextProtosFromParams(parameters url.Values) []string {
	return utils.MultiStringFromParameters(parameters, ParamNextProtos, nil)
}

func ParseUTLS(parameters url.Values, isClient bool) (config *utls.Config, helloId utls.ClientHelloID, e error) {
	config = &utls.Config{
		ServerName: GetSNIFromParams(parameters),
		NextProtos: GetNextProtosFromParams(parameters),
	}

	if isClient {
		helloId, e = GetClientHelloIDFromParams(parameters)
		if e != nil {
			return nil, utls.ClientHelloID{}, e
		}

		verifierFunc, insecure, err := GetCertificatePinningAndInsecure(parameters)
		if err != nil {
			return nil, utls.ClientHelloID{}, err
		}
		config.InsecureSkipVerify = insecure
		config.VerifyPeerCertificate = verifierFunc
	}

	if !isClient {
		clientCaPool, clientCaLen, err := LoadCertPoolFromParams(parameters, ParamClientCA)
		if err != nil {
			return nil, utls.ClientHelloID{}, err
		}
		clientAuth := utls.NoClientCert
		if clientCaLen > 0 {
			clientAuth = utls.RequireAndVerifyClientCert
		}
		config.ClientCAs = clientCaPool
		config.ClientAuth = clientAuth
	}

	certs, err := LoadX509PairsFromParams(parameters)
	if err != nil {
		return nil, utls.ClientHelloID{}, err
	}
	config.Certificates = certs
	return
}

func LoadX509PairsFromParams(parameters url.Values) ([]utls.Certificate, error) {
	certs, keys, err := LoadX509PairsBytesFromParams(parameters)
	if err != nil {
		return nil, err
	}
	pairs := make([]utls.Certificate, len(keys))
	for i := 0; i < len(keys); i++ {
		pairs[i], err = utls.X509KeyPair(certs[i], keys[i])
		if err != nil {
			return nil, err
		}
	}
	return pairs, nil
}

func LoadX509PairFromParams(parameters url.Values) (utls.Certificate, error) {
	cert, key, err := LoadX509PairBytesFromParams(parameters)
	if err != nil {
		var zero utls.Certificate
		return zero, err
	}
	return utls.X509KeyPair(cert, key)
}

func GetClientHelloIDFromParams(parameters url.Values) (utls.ClientHelloID, error) {
	profile, found := utils.GetParameter(parameters, ParamHelloId)
	if found && len(profile) > 0 {
		// let the user define client spec
		profileSplit := strings.Index(profile, MultipleValuesSeparator)
		profileType := profile
		profileVer := ""
		if profileSplit != -1 {
			profileType = profile[:profileSplit]
			profileVer = profile[profileSplit+1:]
		}
		if profileVer == "" {
			profileType = strings.ToLower(profileType)
			switch profileType {
			case "chrome":
				return utls.HelloChrome_Auto, nil
			case "firefox":
				return utls.HelloFirefox_Auto, nil
			case "ios":
				return utls.HelloIOS_Auto, nil
			case "edge":
				return utls.HelloEdge_Auto, nil
			case "android":
				return utls.HelloAndroid_11_OkHttp, nil
			case "safari":
				return utls.HelloSafari_Auto, nil
			case "360":
				return utls.Hello360_Auto, nil
			case "qq":
				return utls.HelloQQ_Auto, nil
			default:
				return utls.ClientHelloID{}, ErrProfileNotSupported
			}
		}
		return utls.ClientHelloID{Client: profileType, Version: profileVer, Seed: nil}, nil
	}
	return utls.HelloGolang, nil
}
