package crypt

import (
	"bytes"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hadi77ir/fragmenter"
	E "github.com/hadi77ir/wsproxy/pkg/errors"
	"github.com/hadi77ir/wsproxy/pkg/utils"
	utls "github.com/refraction-networking/utls"
)

var ErrProfileNotSupported = errors.New("profile not supported by uTLS library")

const (
	ParamSNI                            = "tls.sni"
	ParamNextProtos                     = "tls.alpn"
	ParamDisableALPN                    = "tls.alpn.disable"
	ParamForceHTTP11ALPN                = "tls.alpn.force_http11"
	ParamFragment                       = "tls.fragment"
	ParamHelloId                        = "tls.profile"
	ParamCertificate                    = "tls.cert"
	ParamPrivateKey                     = "tls.key"
	ParamCertificatePin                 = "tls.pin"
	ParamInsecure                       = "tls.insecure"
	ParamCA                             = "tls.ca"
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
	if ShouldDisableALPN(parameters) {
		return nil
	}
	if ShouldForceHTTP11ALPN(parameters) {
		return []string{"http/1.1"}
	}
	nextProtos := utils.MultiStringFromParameters(parameters, ParamNextProtos, nil)
	if len(nextProtos) == 1 && isDisableALPNValue(nextProtos[0]) {
		return nil
	}
	filtered := make([]string, 0, len(nextProtos))
	for _, proto := range nextProtos {
		proto = strings.TrimSpace(proto)
		if proto != "" {
			filtered = append(filtered, proto)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func ShouldDisableALPN(parameters url.Values) bool {
	if value, found := utils.GetParameter(parameters, ParamDisableALPN); found {
		return utils.StrIsTrue(value)
	}
	if value, found := utils.GetParameter(parameters, ParamNextProtos); found {
		return isDisableALPNValue(value)
	}
	return false
}

func ShouldForceHTTP11ALPN(parameters url.Values) bool {
	if value, found := utils.GetParameter(parameters, ParamForceHTTP11ALPN); found {
		return utils.StrIsTrue(value)
	}
	if value, found := utils.GetParameter(parameters, ParamNextProtos); found {
		return strings.EqualFold(strings.TrimSpace(value), "http/1.1")
	}
	return false
}

func isDisableALPNValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "none", "off", "disable", "disabled", "noalpn", "no-alpn":
		return true
	default:
		return false
	}
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
			config.ClientCAs = clientCaPool
		}
		config.ClientAuth = clientAuth
	} else {
		caPool, caPoolLen, err := LoadCertPoolFromParams(parameters, ParamCA)
		if err != nil {
			return nil, utls.ClientHelloID{}, err
		}
		if caPoolLen > 0 {
			config.RootCAs = caPool
		}
	}

	certs, err := LoadX509PairsFromParams(parameters)
	if err != nil {
		return nil, utls.ClientHelloID{}, err
	}
	config.Certificates = certs
	return
}

func GetFragmentConfigFromParams(parameters url.Values) (*fragmenter.FragmentConfig, error) {
	value, found := utils.GetParameter(parameters, ParamFragment)
	if !found || utils.StrIsFalse(value) {
		return nil, nil
	}
	if utils.StrIsTrue(value) || strings.TrimSpace(value) == "" {
		value = "0,1,10,20,0,0"
	}

	config, err := fragmenter.ParseConfig(value)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(value, MultipleValuesSeparator)
	if len(parts) > 5 && strings.TrimSpace(parts[5]) != "" {
		config.IntervalMax, err = time.ParseDuration(strings.TrimSpace(parts[5]))
		if err != nil {
			return nil, fmt.Errorf("invalid max delay: %s", parts[5])
		}
	}
	return config, nil
}

func ApplyALPNPolicy(conn *utls.UConn, parameters url.Values) error {
	switch {
	case ShouldDisableALPN(parameters):
		if conn.ClientHelloID == utls.HelloGolang {
			return nil
		}
		if err := conn.BuildHandshakeState(); err != nil {
			return err
		}
		conn.Extensions = filterALPNExtensions(conn.Extensions)
		return nil
	case ShouldForceHTTP11ALPN(parameters):
		if conn.ClientHelloID == utls.HelloGolang {
			return nil
		}
		if err := conn.BuildHandshakeState(); err != nil {
			return err
		}
		conn.Extensions = forceALPNProtocols(conn.Extensions, []string{"http/1.1"})
		return nil
	default:
		return nil
	}
}

func filterALPNExtensions(extensions []utls.TLSExtension) []utls.TLSExtension {
	filtered := make([]utls.TLSExtension, 0, len(extensions))
	for _, ext := range extensions {
		switch ext.(type) {
		case *utls.ALPNExtension, *utls.ApplicationSettingsExtension:
			continue
		default:
			filtered = append(filtered, ext)
		}
	}
	return filtered
}

func forceALPNProtocols(extensions []utls.TLSExtension, protocols []string) []utls.TLSExtension {
	found := false
	for i, ext := range extensions {
		if _, ok := ext.(*utls.ALPNExtension); ok {
			extensions[i] = &utls.ALPNExtension{AlpnProtocols: protocols}
			found = true
			continue
		}
	}
	if !found {
		extensions = append(extensions, &utls.ALPNExtension{AlpnProtocols: protocols})
	}
	return filterApplicationSettingsExtensions(extensions)
}

func filterApplicationSettingsExtensions(extensions []utls.TLSExtension) []utls.TLSExtension {
	filtered := make([]utls.TLSExtension, 0, len(extensions))
	for _, ext := range extensions {
		if _, ok := ext.(*utls.ApplicationSettingsExtension); ok {
			continue
		}
		filtered = append(filtered, ext)
	}
	return filtered
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
		if helloID, ok := clientHelloIDs[normalizeProfile(profile)]; ok {
			return helloID, nil
		}

		profileType, profileVer, ok := splitProfile(profile)
		if !ok {
			return utls.ClientHelloID{}, ErrProfileNotSupported
		}
		canonicalClient, ok := clientHelloClients[normalizeProfile(profileType)]
		if !ok {
			return utls.ClientHelloID{}, ErrProfileNotSupported
		}
		return utls.ClientHelloID{Client: canonicalClient, Version: profileVer, Seed: nil}, nil
	}
	return utls.HelloGolang, nil
}

func splitProfile(profile string) (profileType string, profileVer string, ok bool) {
	profile = strings.TrimSpace(profile)
	for _, sep := range []string{MultipleValuesSeparator, ":", "/", "_", "-"} {
		if i := strings.LastIndex(profile, sep); i > 0 && i < len(profile)-len(sep) {
			return profile[:i], profile[i+len(sep):], true
		}
	}
	return "", "", false
}

func normalizeProfile(profile string) string {
	profile = strings.TrimSpace(strings.ToLower(profile))
	profile = strings.TrimPrefix(profile, "hello")
	profile = strings.ReplaceAll(profile, "browser", "")
	profile = strings.ReplaceAll(profile, ".", "")
	profile = strings.ReplaceAll(profile, "_", "")
	profile = strings.ReplaceAll(profile, "-", "")
	profile = strings.ReplaceAll(profile, "/", "")
	profile = strings.ReplaceAll(profile, ",", "")
	profile = strings.ReplaceAll(profile, ":", "")
	return profile
}

var clientHelloClients = map[string]string{
	"360":        "360Browser",
	"360browser": "360Browser",
	"android":    "Android",
	"chrome":     "Chrome",
	"edge":       "Edge",
	"firefox":    "Firefox",
	"ios":        "iOS",
	"qq":         "QQBrowser",
	"qqbrowser":  "QQBrowser",
	"safari":     "Safari",
}

var clientHelloIDs = map[string]utls.ClientHelloID{
	"360":                   utls.Hello360_Auto,
	"360auto":               utls.Hello360_Auto,
	"36075":                 utls.Hello360_7_5,
	"360110":                utls.Hello360_11_0,
	"android":               utls.HelloAndroid_11_OkHttp,
	"android11":             utls.HelloAndroid_11_OkHttp,
	"android11okhttp":       utls.HelloAndroid_11_OkHttp,
	"chrome":                utls.HelloChrome_Auto,
	"chromeauto":            utls.HelloChrome_Auto,
	"chrome58":              utls.HelloChrome_58,
	"chrome62":              utls.HelloChrome_62,
	"chrome70":              utls.HelloChrome_70,
	"chrome72":              utls.HelloChrome_72,
	"chrome83":              utls.HelloChrome_83,
	"chrome87":              utls.HelloChrome_87,
	"chrome96":              utls.HelloChrome_96,
	"chrome100":             utls.HelloChrome_100,
	"chrome102":             utls.HelloChrome_102,
	"chrome106":             utls.HelloChrome_106_Shuffle,
	"chrome106shuffle":      utls.HelloChrome_106_Shuffle,
	"custom":                utls.HelloCustom,
	"edge":                  utls.HelloEdge_Auto,
	"edgeauto":              utls.HelloEdge_Auto,
	"edge85":                utls.HelloEdge_85,
	"edge106":               utls.HelloEdge_106,
	"firefox":               utls.HelloFirefox_Auto,
	"firefoxauto":           utls.HelloFirefox_Auto,
	"firefox55":             utls.HelloFirefox_55,
	"firefox56":             utls.HelloFirefox_56,
	"firefox63":             utls.HelloFirefox_63,
	"firefox65":             utls.HelloFirefox_65,
	"firefox99":             utls.HelloFirefox_99,
	"firefox102":            utls.HelloFirefox_102,
	"firefox105":            utls.HelloFirefox_105,
	"golang":                utls.HelloGolang,
	"go":                    utls.HelloGolang,
	"ios":                   utls.HelloIOS_Auto,
	"iosauto":               utls.HelloIOS_Auto,
	"ios111":                utls.HelloIOS_11_1,
	"ios121":                utls.HelloIOS_12_1,
	"ios13":                 utls.HelloIOS_13,
	"ios14":                 utls.HelloIOS_14,
	"qq":                    utls.HelloQQ_Auto,
	"qqauto":                utls.HelloQQ_Auto,
	"qq111":                 utls.HelloQQ_11_1,
	"random":                utls.HelloRandomized,
	"randomized":            utls.HelloRandomized,
	"randomizedalpn":        utls.HelloRandomizedALPN,
	"randomizednoalpn":      utls.HelloRandomizedNoALPN,
	"randomnoalpn":          utls.HelloRandomizedNoALPN,
	"randomizedwithoutalpn": utls.HelloRandomizedNoALPN,
	"safari":                utls.HelloSafari_Auto,
	"safariauto":            utls.HelloSafari_Auto,
	"safari160":             utls.HelloSafari_16_0,
}
