{{/*
Expand the name of the chart.
*/}}
{{- define "llmos-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "llmos-operator.fullname" -}}
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
{{- define "llmos-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "llmos-operator.labels" -}}
helm.sh/chart: {{ include "llmos-operator.chart" . }}
{{ include "llmos-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Webhook labels
*/}}
{{- define "llmos-operator.webhookLabels" -}}
{{ include "llmos-operator.labels" . }}
app.llmos.ai/webhook: "true"
{{- end }}

{{/*
System charts labels
*/}}
{{- define "llmos-operator.systemChartsLabels" -}}
{{ include "llmos-operator.labels" . }}
app.llmos.ai/system-charts: "true"
{{- end }}

{{/*
Selector labels
*/}}
{{- define "llmos-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "llmos-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Webhook selector labels
*/}}
{{- define "llmos-operator.webhookSelectorLabels" -}}
{{ include "llmos-operator.selectorLabels" . }}
app.llmos.ai/webhook: "true"
{{- end }}

{{/*
System charts selector labels
*/}}
{{- define "llmos-operator.systemChartsSelectorLabels" -}}
{{ include "llmos-operator.selectorLabels" . }}
app.llmos.ai/system-charts: "true"
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "llmos-operator.serviceAccountName" -}}
{{- if .Values.operator.apiserver.serviceAccount.create }}
{{- default (include "llmos-operator.fullname" .) .Values.operator.apiserver.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.operator.apiserver.serviceAccount.name }}
{{- end }}
{{- end }}
