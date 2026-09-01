{{/*
Expand the name of the chart.
*/}}
{{- define "roboto-go.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "roboto-go.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "roboto-go.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "roboto-go.labels" -}}
helm.sh/chart: {{ include "roboto-go.chart" . }}
{{ include "roboto-go.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
ollama-config-sha: {{ .Values.ollama | toJson | sha256sum | trunc 63 }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "roboto-go.selectorLabels" -}}
app.kubernetes.io/name: {{ include "roboto-go.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
labels - lavalink
*/}}
{{- define "roboto-go.labelsLavalink" -}}
helm.sh/chart: {{ include "roboto-go.chart" . }}
{{ include "roboto-go.selectorLabelsLavalink" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels lavalink
*/}}
{{- define "roboto-go.selectorLabelsLavalink" -}}
app.kubernetes.io/name: {{ include "roboto-go.name" . }}-lavalink
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "roboto-go.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "roboto-go.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
autogenerate very secure, totally secret password :)
*/}}
{{- define "roboto-go.lavalinkPassword" -}}
password123
{{- end }}

{{/*
adjust lavalink config with variables from helm
*/}}
{{- define "roboto-go.lavalinkConfig" }}
{{- $config := .Values.lavalink.config }}
{{- /* override remote-ciphers */}}
{{- if .Values.ytCipher.enabled }}
{{- $remoteCipherConfig := dig "plugins" "youtube" "remoteCipher" (dict) $config }}
{{- $_ := set $remoteCipherConfig "url" (printf "http://%s-yt-cipher.%s.svc.cluster.local:%s" ( include "roboto-go.name" .) .Release.Namespace (.Values.ytCipher.service.port | toString)) }}
{{- $override := dict "plugins" (dict "youtube" ( dict "remoteCipher" $remoteCipherConfig) ) }}
{{- $config = merge $config $override }}
{{- end }}
{{- /* return config */}}
{{- $config | toYaml }}
{{- end }}


{{/*
labels - yt-cipher
*/}}
{{- define "roboto-go.labelsYtCipher" -}}
helm.sh/chart: {{ include "roboto-go.chart" . }}
{{ include "roboto-go.selectorLabelsYtCipher" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels yt-cipher
*/}}
{{- define "roboto-go.selectorLabelsYtCipher" -}}
app.kubernetes.io/name: {{ include "roboto-go.name" . }}-yt-cipher
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
