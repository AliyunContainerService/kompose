package compose

import (
	"fmt"
	"github.com/flynn/go-shlex"
	"github.com/kubernetes/kompose/pkg/kobject"
	"github.com/pkg/errors"
	"k8s.io/kubernetes/pkg/api"
	"net"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const DEFAULT_ROOT_DOMAIN = "alicontainer.com"

// Support Aliyun Extension Labels
const EXT_LABEL_PREFIX = "aliyun"
const LABEL_SERVICE_SCALE = EXT_LABEL_PREFIX + ".scale"
const LABEL_SERVICE_LOGS = EXT_LABEL_PREFIX + ".logs"
const LABEL_SERVICE_GLOBAL = EXT_LABEL_PREFIX + ".global"
const LABEL_SERVICE_DEPENDS = EXT_LABEL_PREFIX + ".depends"

const LABEL_SERVICE_ROUTING_SEPARATOR = ","
const LABEL_SERVICE_ROUTING_PREFIX = EXT_LABEL_PREFIX + ".routing"
const LABEL_SERVICE_ROUTING_SESSION = LABEL_SERVICE_ROUTING_PREFIX + ".session_sticky"
const LABEL_SERVICE_ROUTING_WEIGHT = LABEL_SERVICE_ROUTING_PREFIX + ".weight"
const LABEL_SERVICE_ROUTING_SSL_CERTS = LABEL_SERVICE_ROUTING_PREFIX + ".ssl_cert"
const LABEL_SERVICE_ROUTING_EXTRA_SETTINGS = LABEL_SERVICE_ROUTING_PREFIX + ".extra_setting"
const LABEL_SERVICE_ROUTING_TIMEOUT_SETTINGS = LABEL_SERVICE_ROUTING_PREFIX + ".timeout_setting"
const LABEL_SERVICE_ROUTING_WORKER_NUMBER = LABEL_SERVICE_ROUTING_PREFIX + ".worker"
const LABEL_SERVICE_ROUTING_PORT_PREFIX = LABEL_SERVICE_ROUTING_PREFIX + ".port_"

const LABEL_SERVICE_LB_PORT_PREFIX = EXT_LABEL_PREFIX + ".lb.port_"
const LABEL_SERVICE_VIP_PORT_PREFIX = EXT_LABEL_PREFIX + ".vip.port_"

// Probe Labels
const LABEL_SERVICE_PROBE_URL = EXT_LABEL_PREFIX + ".probe.url"
const LABEL_SERVICE_PROBE_CMD = EXT_LABEL_PREFIX + ".probe.cmd"
const LABEL_SERVICE_PROBE_INIT_DELAY_SECONDS = EXT_LABEL_PREFIX + ".probe.initial_delay_seconds"
const LABEL_SERVICE_PROBE_TIMEOUT_SECONDS = EXT_LABEL_PREFIX + ".probe.timeout_seconds"

// gpu labels
const LABEL_GPU = EXT_LABEL_PREFIX + ".gpu"

// sls labels
const LABEL_SLS_PREFIX = EXT_LABEL_PREFIX + ".log_store_"
const LABEL_SLS_PROJECT = EXT_LABEL_PREFIX + ".log_project"
const LABEL_SLS_TTL_PATTERN = EXT_LABEL_PREFIX + ".log_ttl_%s"

// monitoring labels
const LABEL_MONITORING_PREFIX = EXT_LABEL_PREFIX + ".monitoring"
const LABEL_MONITORING_CONF = LABEL_MONITORING_PREFIX + ".conf"
const LABEL_MONITORING_HTTP = LABEL_MONITORING_PREFIX + ".http"
const LABEL_MONITORING_SCRIPT = LABEL_MONITORING_PREFIX + ".script"
const LABEL_MONITORING_RELOAD = LABEL_MONITORING_PREFIX + ".reload"
const LABEL_MONITORING_TAGS = LABEL_MONITORING_PREFIX + ".tags"
const LABEL_MONITORING_FORMAT = LABEL_MONITORING_PREFIX + ".format"
const LABEL_MONITORING_INTERVAL = LABEL_MONITORING_PREFIX + ".interval"

const LABEL_MONITORING_ADDON_PREFIX = LABEL_MONITORING_PREFIX + ".addon"
const LABEL_MONITORING_ADDON_INFLUXDB = LABEL_MONITORING_ADDON_PREFIX + ".influxdb"
const LABEL_MONITORING_ADDON_INFLUXDB_RETENTION_POLICY = LABEL_MONITORING_ADDON_PREFIX + ".influxdb_retention_policy"
const LABEL_MONITORING_ADDON_PROMETHEUS = LABEL_MONITORING_ADDON_PREFIX + ".prometheus"

// auto-scaling labels
const LABEL_AUTO_SCALING_PREFIX = EXT_LABEL_PREFIX + ".auto_scaling"
const LABEL_AUTO_SCALING_MAX_CPU = LABEL_AUTO_SCALING_PREFIX + ".max_cpu"
const LABEL_AUTO_SCALING_MIN_CPU = LABEL_AUTO_SCALING_PREFIX + ".min_cpu"
const LABEL_AUTO_SCALING_MAX_MEMORY = LABEL_AUTO_SCALING_PREFIX + ".max_memory"
const LABEL_AUTO_SCALING_MIN_MEMORY = LABEL_AUTO_SCALING_PREFIX + ".min_memory"
const LABEL_AUTO_SCALING_MAX_INTERNET_INRATE = LABEL_AUTO_SCALING_PREFIX + ".max_internetInRate"
const LABEL_AUTO_SCALING_MIN_INTERNET_INRATE = LABEL_AUTO_SCALING_PREFIX + ".min_internetInRate"
const LABEL_AUTO_SCALING_MAX_INTERNET_OUTRATE = LABEL_AUTO_SCALING_PREFIX + ".max_internetOutRate"
const LABEL_AUTO_SCALING_MIN_INTERNET_OUTRATE = LABEL_AUTO_SCALING_PREFIX + ".min_internetOutRate"

const LABEL_AUTO_SCALING_MAX_INSTANCES = LABEL_AUTO_SCALING_PREFIX + ".max_instances"
const LABEL_AUTO_SCALING_MIN_INSTANCES = LABEL_AUTO_SCALING_PREFIX + ".min_instances"
const LABEL_AUTO_SCALING_STEP = LABEL_AUTO_SCALING_PREFIX + ".step"
const LABEL_AUTO_SCALING_PERIOD = LABEL_AUTO_SCALING_PREFIX + ".period"

const ANNOTATION_LOADBALANCER_PREFIX = "service.beta.kubernetes.io/alicloud-loadbalancer-"
const ANNOTATION_LOADBALANCER_ID = ANNOTATION_LOADBALANCER_PREFIX + "id"
const ANNOTATION_LOADBALANCER_PROTOCOL_PORT = ANNOTATION_LOADBALANCER_PREFIX + "protocol-port"

// This function will retrieve volumes for each service, as well as it will parse volume information and store it in Volumes struct
func handleAliyunExt(svcName string, service *kobject.ServiceConfig) {
	for key, value := range service.Labels {

		value = strings.TrimSpace(value)
		var err error

		switch key {

		case LABEL_SERVICE_SCALE:
			service.Replicas, err = strconv.Atoi(value)
		case LABEL_SERVICE_GLOBAL:
			if strings.ToLower(value) == "true" {
				service.DeployMode = "global"
			}
		case LABEL_SERVICE_PROBE_URL:
			err = parseServiceProbe(value, &service.HealthChecks)
		case LABEL_SERVICE_PROBE_CMD:
			service.HealthChecks.Test, err = shlex.Split(value)
		case LABEL_SERVICE_PROBE_TIMEOUT_SECONDS:
			service.HealthChecks.Timeout, err = parseProbeSeconds(value)
		case LABEL_SERVICE_PROBE_INIT_DELAY_SECONDS:
			service.HealthChecks.StartPeriod, err = parseProbeSeconds(value)
		case LABEL_GPU:
			service.GPUs, err = strconv.Atoi(value)
		default:
			if strings.HasPrefix(key, LABEL_SERVICE_LB_PORT_PREFIX) {
				service.ServiceType = string(api.ServiceTypeLoadBalancer)
				service.ServiceAnnotations, err = parseLbs(key, value)
			}

			if strings.HasPrefix(key, LABEL_SERVICE_ROUTING_PORT_PREFIX) {
				var routings []Routing
				routings, err = parseRouting(key, value)
				if err == nil && len(routings) > 0 {
					service.ExposeService = routings[0].VirtualHost
					service.ExposeServicePath = routings[0].ContextRoot
				}
			}

			if strings.HasPrefix(key, LABEL_SLS_PREFIX) {
				var logConfig *LogConfig
				logConfig, err = parseLogConfig(key, value)
				if err == nil {
					service.Environment = append(service.Environment, kobject.EnvVar{
						Name:  "aliyun_logs_" + logConfig.Name,
						Value: value,
					})
					if logConfig.Type == "file" {
						// Add mount point
						if service.LogVolumes == nil {
							service.LogVolumes = make(map[string]string)
						}
						service.LogVolumes[logConfig.Name] = logConfig.ContainerPath
					}
				}
			}
		}

		if err != nil {
			errors.Wrap(err, fmt.Sprint("invalid value for %s label: %s", key, value))
		}
	}
}

const _CONTAINER = "container"

func parseServiceProbe(value string, probe *kobject.HealthCheck) error {

	var err error
	uri, err := url.Parse(value)
	if err != nil {
		return err
	}
	switch uri.Scheme {
	case "http":
		host := uri.Host
		portStr := "80"
		port := 80
		if strings.Index(uri.Host, ":") >= 0 {
			host, portStr, err = net.SplitHostPort(uri.Host)
		}
		if portStr != "" {
			port, err = strconv.Atoi(portStr)
		}
		if err == nil {
			if host != _CONTAINER {
				err = fmt.Errorf("Host of HTTP probe URL should be container")
			} else {
				action := kobject.HTTPGetAction{
					Port: port,
					Path: uri.Path,
				}
				probe.HTTPGet = &action
			}
		}
	case "tcp":
		host, portStr, err2 := net.SplitHostPort(uri.Host)
		err := err2
		port := 0
		if portStr == "" {
			err = fmt.Errorf("Host of HTTP probe URL should be container")
		} else if host != _CONTAINER {
			err = fmt.Errorf("Host of TCP probe should be container")
		} else {
			port, err = strconv.Atoi(portStr)
		}
		if err == nil {
			action := kobject.TCPSocketAction{
				Port: port,
			}
			probe.TCPSocket = &action
		}
	default:
		err = fmt.Errorf("Unsupported scheme of probe URL: %s", value)
	}
	return err
}

func parseProbeSeconds(value string) (int32, error) {
	result, err := strconv.Atoi(value)
	return int32(result), err
}

type Routing struct {
	VirtualHost string `json:"virtual_host,omitempty"`
	ContextRoot string `json:"context_root,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
	Port        int    `json:"port,omitempty"`
}

type Port struct {
	FrontendPort int    `json:"frontend_port,omitempty"`
	BackendPort  int    `json:"backend_port,omitempty"`
	Protocol     string `json:"protocol,omitempty"`
}

type SlbInfo struct {
	SlbName       string `json:"slb_name, omitempty"`
	SlbId         string `json:"slb_id, omitempty"`
	FrontendPort  int    `json:"frontend_port, omitempty"`
	BackendPort   int    `json:"backend_port, omitempty"`
	Protocol      string `json:"protocol,omitempty"`
	ContainerPort int    `json:"container_port,omitempty"`
}

//func validateSlbLabels(labels map[string]string) (slbInfo []*SlbInfo, err error) {
//	slbInfo = make([]*SlbInfo, 0)
//	for key, labelContent := range labels {
//		if strings.HasPrefix(key, LABEL_SERVICE_LB_PORT_PREFIX) {
//			lbs, err := ParseLbs(key, labelContent)
//			if err != nil {
//				return nil, err
//			} else {
//				slbInfo = append(slbInfo, lbs...)
//			}
//		}
//	}
//	return slbInfo, nil
//}
//

func parseLbs(key, value string) (annotations map[string]string, err error) {
	slbInfoList := make([]*SlbInfo, 0)
	annotations = make(map[string]string)
	slbLabels := strings.Split(value, ";")
	for _, slbLabel := range slbLabels {
		slbLabel = strings.TrimSpace(slbLabel)
		if slbLabel != "" {
			slbInfo, err := parseLb(key, slbLabel)
			if err != nil {
				return nil, err
			}
			slbInfoList = append(slbInfoList, slbInfo)
		}
	}

	if len(slbInfoList) == 0 {
		return nil, nil
	}

	annotations[ANNOTATION_LOADBALANCER_ID] = slbInfoList[0].SlbName
	annotations[ANNOTATION_LOADBALANCER_PROTOCOL_PORT] = fmt.Sprintf("%s:%d", slbInfoList[0].Protocol, slbInfoList[0].FrontendPort)

	return annotations, nil
}

func parseLb(key, value string) (*SlbInfo, error) {
	port, err := getNatContainerPort(key)
	if err != nil {
		return nil, err
	}
	slbInfo := &SlbInfo{}
	slbInfo.ContainerPort = port
	if err := validUri(value); err != nil {
		return nil, fmt.Errorf("invalid loadbalancer uri %s: %v", value, err)
	}

	uri, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("failed to parse loadbalancer uri %s: %v", value, err)
	}
	host := strings.Split(uri.Host, ":")
	slbInfo.SlbName = host[0]
	if len(host) > 1 {
		slbInfo.FrontendPort, err = strconv.Atoi(host[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse loadbalancer frountent port %s: %v", host[1], err)
		}
	} else {
		slbInfo.FrontendPort = 80
	}
	slbInfo.Protocol = uri.Scheme
	if err = assertSlbInfoSyntaxValid(slbInfo); err != nil {
		return nil, err
	}
	return slbInfo, nil

}

const _VALID_VIP_TEMPLATE = `^\b((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.|$)){4}\b$`
const _VALID_SLB_NAME_TEMPLATE = `^\b[a-zA-Z0-9-_\/.]{1,80}\b$`
const _VALID_SLB_NAME_LENGTH = 80
const _VALID_URI = `^[a-z]{3,5}://[a-zA-Z0-9-_\/.]{1,80}:[0-9]{1,5}$`

func validUri(uri string) error {
	if match, _ := regexp.MatchString(_VALID_URI, uri); match {
		return nil
	}
	return fmt.Errorf("the uri: %s is not valid", uri)
}

func validVip(vip string) error {
	if match, _ := regexp.MatchString(_VALID_VIP_TEMPLATE, vip); match {
		return nil
	}
	return fmt.Errorf("the vip: %s is not valid", vip)
}

func assertSlbInfoSyntaxValid(slbInfo *SlbInfo) error {
	if slbInfo.Protocol != "udp" && slbInfo.Protocol != "tcp" && slbInfo.Protocol != "http" && slbInfo.Protocol != "https" {
		return fmt.Errorf("invalid loadbalancer protocol: %s", slbInfo.Protocol)
	}

	if slbInfo.ContainerPort < 1 || slbInfo.ContainerPort > 65535 {
		return fmt.Errorf("invalid container port for loadbalancer config: %d", slbInfo.ContainerPort)
	}

	if slbInfo.FrontendPort < 1 || slbInfo.FrontendPort > 65535 {
		return fmt.Errorf("invalid frontend port for loadbalancer config: %v", slbInfo.FrontendPort)

	}

	// 长度限制为1-80个字符，允许包含字母、数字、'-'、'/'、'.'、'_'这些字符
	if err := validSlbName(slbInfo.SlbName); err != nil {
		return err
	}
	return nil
}

func validSlbName(slbName string) error {
	if len(slbName) < _VALID_SLB_NAME_LENGTH {
		if match, _ := regexp.MatchString(_VALID_SLB_NAME_TEMPLATE, slbName); match {
			return nil
		}
		return fmt.Errorf("invalid name of loadbalancer: %s", slbName)
	} else {
		return fmt.Errorf("the name of loadbalancer is too long: %s", slbName)
	}
}

func parseRouting(key, value string) ([]Routing, error) {
	urls := strings.Split(value, ";")
	port, err := getRoutingContainerPort(key)
	if err != nil {
		return nil, err
	}
	routings := []Routing{}
	for _, urlOrUrlPrefix := range urls {
		if urlOrUrlPrefix != "" {
			uri, err := normalizeUrl(urlOrUrlPrefix)
			if err != nil {
				return nil, err
			}
			singleRouting := Routing{}
			singleRouting.Port = port
			singleRouting.Protocol = uri.Scheme
			fields := strings.Split(uri.Host, ".")
			if len(fields) == 1 {
				virtualHost, err := getOfferedVirtaulHost(uri)
				if err != nil {
					return nil, err
				} else {
					singleRouting.VirtualHost = virtualHost
				}
			} else {
				err = validateUserVirtualHost(uri.Host)
				if err != nil {
					return nil, err
				} else {
					singleRouting.VirtualHost = uri.Host
				}
			}
			singleRouting.ContextRoot = uri.Path
			if err := checkRoutingDuplicate(routings, singleRouting); err != nil {
				return nil, err
			} else {
				routings = append(routings, singleRouting)
			}
		}
	}
	return routings, validateRoutings(routings)
}

func validateRoutings(routings []Routing) error {
	for _, routing := range routings {
		if err := validRoutingProtocol(routing.Protocol); err != nil {
			return err
		}
		if err := ValidDomain(routing.VirtualHost); err != nil {
			return err
		}
		if err := validContextRoot(routing.ContextRoot); err != nil {
			return err
		}
	}
	return nil
}

const _VALID_DOMAIN_LENGTH = 128
const _VALID_CONTEXT_ROOT_LENGTH = 128
const _VALID_DOMAIN_TEMPLATE = `^\b([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,63}\b$`
const _VALID_CONTEXT_ROOT_TEMPLATE = `^/[a-z0-9-_\/.]+$`

func validRoutingProtocol(protocol string) (err error) {
	if "ws" != protocol && "wss" != protocol && "http" != protocol && "https" != protocol && "tcp" != protocol {
		return fmt.Errorf("invalid protocol: %s", protocol)
	}
	return nil
}

func ValidDomain(domain string) (err error) {
	if len(domain) < _VALID_DOMAIN_LENGTH {
		domain = strings.ToLower(domain)
		if match, _ := regexp.MatchString(_VALID_DOMAIN_TEMPLATE, domain); match {
			return nil
		}
		return fmt.Errorf("invalid domain: %s", domain)
	} else {
		return fmt.Errorf("the domain is too long: %s", domain)
	}
}

func validContextRoot(contextRoot string) (err error) {
	if contextRoot == "" || contextRoot == "/" {
		return nil
	}
	if len(contextRoot) < _VALID_CONTEXT_ROOT_LENGTH {
		contextRoot = strings.ToLower(contextRoot)
		if match, _ := regexp.MatchString(_VALID_CONTEXT_ROOT_TEMPLATE, contextRoot); match {
			return nil
		} else {
			return fmt.Errorf("invalid context root: %s", contextRoot)
		}
	} else {
		return fmt.Errorf("the context root is too long: %s", contextRoot)
	}
}

func normalizeUrl(urlOrUrlPrefix string) (uri *url.URL, err error) {
	// default to http:// scheme
	if !strings.HasPrefix(urlOrUrlPrefix, "http://") &&
		!strings.HasPrefix(urlOrUrlPrefix, "https://") &&
		!strings.HasPrefix(urlOrUrlPrefix, "ws://") &&
		!strings.HasPrefix(urlOrUrlPrefix, "wss://") &&
		!strings.HasPrefix(urlOrUrlPrefix, "tcp://") {
		urlOrUrlPrefix = "http://" + urlOrUrlPrefix
	}
	uri, err = url.Parse(urlOrUrlPrefix)
	if err != nil {
		err := fmt.Errorf("invalid routing url: %s error: %s", urlOrUrlPrefix, err.Error())
		return nil, err
	}
	if uri.Host == "" {
		return nil, fmt.Errorf("host name in routing uri is missing: %s", urlOrUrlPrefix)
	}
	return uri, nil
}

func GetRootDomain() string {
	return "your-cluster-id." + DEFAULT_ROOT_DOMAIN
}

func getOfferedVirtaulHost(uri *url.URL) (virtualHost string, err error) {
	reservedDomainSuffix := GetRootDomain()

	if uri.Scheme == "https" {
		err := fmt.Errorf("https scheme, i.e. %s://%s must have fully qualified domain name", uri.Scheme, uri.Host)
		return "", err
	}
	return uri.Host + "." + reservedDomainSuffix, nil
}

func validateUserVirtualHost(uriHost string) (err error) {
	reservedDomainSplits := strings.Split(GetRootDomain(), ".")
	if len(reservedDomainSplits) < 2 {
		return fmt.Errorf("root domain:%v only has one class, please ask the administrator for help", GetRootDomain())
	}
	if strings.HasSuffix(uriHost, DEFAULT_ROOT_DOMAIN) {
		return fmt.Errorf("%s is not allowed as user provided domain", uriHost)
	}
	return nil
}

func checkRoutingDuplicate(routings []Routing, routing Routing) (err error) {
	for _, route := range routings {
		if (route.VirtualHost == routing.VirtualHost) && (route.ContextRoot == routing.ContextRoot) &&
			(route.Protocol == routing.Protocol) {
			return fmt.Errorf("found duplicated routing %s://%s/%s",
				route.Protocol, route.VirtualHost, route.ContextRoot)
		}
	}
	return nil
}

func getRoutingContainerPort(key string) (port int, err error) {
	port, err = strconv.Atoi(key[len(LABEL_SERVICE_ROUTING_PORT_PREFIX):])

	if err != nil {
		return 0, fmt.Errorf("failed to parse routing port: %s", key, err)
	}

	if port < 1 || port > 65535 {
		err = fmt.Errorf("invalid routing port number: %s", key)
		return 0, err
	}
	return port, nil
}

func getNatContainerPort(key string) (port int, err error) {
	port, err = strconv.Atoi(key[len(LABEL_SERVICE_LB_PORT_PREFIX):])
	if err != nil {
		return 0, fmt.Errorf("failed to parse loadbalancer port: %s", key, err)
	}

	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid loadbalancer port number: %s", key)
	}
	return port, nil
}

type LogConfig struct {
	Name          string `json:"name,omitempty"`
	ContainerPath string `json:"containerPath,omitempty"`
	FilePattern   string `json:"filePattern,omitempty"`
	Type          string `json:"type,omitempty"`
	TTL           string `json:"ttl,omitempty"`
}

//
//type SLSInfo struct {
//	Project    string `json:"project,omitempty"`
//	Integrated bool   `json:"integrated,omitempty"`
//}

func parseLogConfig(key, path string) (*LogConfig, error) {

	var (
		containerPath string
		filePattern   string
		typ           string
	)
	path = strings.TrimSpace(path)
	name := key[len(LABEL_SLS_PREFIX):]
	if len(name) == 0 {
		return nil, fmt.Errorf("invalid SLS label key: %s", key)
	}

	if path != "stdout" && !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("invalid log path: %s, must be stdout or start with /", path)
	}

	if path == "stdout" {
		filePattern = "stdout"
		containerPath = ""
		typ = "stdout"
	} else {
		filePattern = filepath.Base(path)
		containerPath = filepath.Dir(path)
		typ = "file"
	}

	return &LogConfig{
		Name:          name,
		ContainerPath: containerPath,
		FilePattern:   filePattern,
		Type:          typ,
	}, nil
}
