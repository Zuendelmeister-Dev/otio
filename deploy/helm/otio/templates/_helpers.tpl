{{- define "otio.prefix" -}}
{{- if gt (len .Release.Name) 40 -}}{{ fail "Use a release name of at most 40 characters" }}{{- end -}}
{{- .Release.Name -}}
{{- end -}}

{{- define "otio.amqpURL" -}}
{{- printf "amqp://%s:%s@%s-rabbitmq:5672/" (.Values.credentials.rabbitmqUser | urlquery | replace "+" "%20") (.Values.credentials.rabbitmqPassword | urlquery | replace "+" "%20") (include "otio.prefix" .) -}}
{{- end -}}

{{- define "otio.senseConfig" -}}
{{- $config := .Files.Get "files/sense.json" | fromJson -}}
{{- $prefix := include "otio.prefix" . -}}
{{- $_ := set $config.broker "host" (printf "%s-mqtt" $prefix) -}}
{{- range $source := $config.sources -}}
  {{- $host := $source.host -}}
  {{- $_ := set $source "host" (printf "%s-%s" $prefix $host) -}}
  {{- $_ := set $source.options "gatewayURL" (printf "http://%s-protocol-lab:8500" $prefix) -}}
  {{- $_ := set $source.options "connection" (replace (printf "://%s:" $host) (printf "://%s-%s:" $prefix $host) $source.options.connection) -}}
  {{- if eq $source.type "lab-amqp" -}}{{- $_ := set $source.options "connection" (include "otio.amqpURL" $) -}}{{- end -}}
{{- end -}}
{{- $config | toPrettyJson -}}
{{- end -}}

{{- define "otio.topology" -}}
{{- $topology := .Files.Get "files/topology.json" | fromJson -}}
{{- $prefix := include "otio.prefix" . -}}
{{- range $sense := $topology.senses -}}
  {{- $_ := set $sense "internalUrl" (printf "http://%s-sense:8100" $prefix) -}}
  {{- $_ := set $sense "url" $.Values.browser.senseURL -}}
{{- end -}}
{{- range $sim := $topology.presenses -}}
  {{- $_ := set $sim "internalUrl" (printf "http://%s-protocol-lab:8500" $prefix) -}}
  {{- $_ := set $sim "url" $.Values.browser.protocolLabURL -}}
{{- end -}}
{{- range $agent := $topology.agents -}}
  {{- $_ := set $agent "sourceHost" (printf "%s-%s" $prefix $agent.sourceHost) -}}
  {{- $_ := set $agent "host" $agent.sourceHost -}}
{{- end -}}
{{- $topology | toPrettyJson -}}
{{- end -}}
