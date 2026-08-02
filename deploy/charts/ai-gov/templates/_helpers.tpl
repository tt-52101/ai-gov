{{/*
展开 Chart 名称。
*/}}
{{- define "ai-gov.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
创建默认的完全限定应用名称。
按照 DNS 命名规范截断至 63 字符。
如果 release 名称包含 chart 名称，则使用 release 名称作为全名。
*/}}
{{- define "ai-gov.fullname" -}}
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
创建 chart 名称和版本，用于 chart 标签。
*/}}
{{- define "ai-gov.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
通用标签。
*/}}
{{- define "ai-gov.labels" -}}
helm.sh/chart: {{ include "ai-gov.chart" . }}
{{ include "ai-gov.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
选择器标签。
*/}}
{{- define "ai-gov.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ai-gov.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
创建服务账号名称。
*/}}
{{- define "ai-gov.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "ai-gov.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
API 服务器容器名称。
*/}}
{{- define "ai-gov.apiContainerName" -}}
api-server
{{- end }}

{{/*
UI 服务器容器名称。
*/}}
{{- define "ai-gov.uiContainerName" -}}
ui-server
{{- end }}

{{/*
镜像标签。
*/}}
{{- define "ai-gov.imageTag" -}}
{{- default .Chart.AppVersion .Values.image.tag }}
{{- end }}