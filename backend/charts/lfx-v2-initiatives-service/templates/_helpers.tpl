{{- /*
Copyright The Linux Foundation and each contributor to LFX.
SPDX-License-Identifier: MIT
*/ -}}
{{/*
Expand the name of the chart.
*/}}
{{- define "lfx-v2-initiatives-service.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "lfx-v2-initiatives-service.fullname" -}}
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
Create chart label.
*/}}
{{- define "lfx-v2-initiatives-service.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "lfx-v2-initiatives-service.labels" -}}
helm.sh/chart: {{ include "lfx-v2-initiatives-service.chart" . }}
{{ include "lfx-v2-initiatives-service.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "lfx-v2-initiatives-service.selectorLabels" -}}
app.kubernetes.io/name: {{ include "lfx-v2-initiatives-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
ServiceAccount name.
*/}}
{{- define "lfx-v2-initiatives-service.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "lfx-v2-initiatives-service.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
