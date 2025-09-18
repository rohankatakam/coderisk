"""
Configuration Risk Detector

Analyzes risky changes to Kubernetes, Terraform, YAML, and other configuration files.
Focuses on resource limits, security settings, and deployment configurations.

Time budget: 10-40ms
"""

import re
import yaml
import json
from typing import Dict, List, Set, Optional, Any
from pathlib import Path

from . import BaseDetector, DetectorResult, ChangeContext, FileChange, detector_registry


@detector_registry.register
class ConfigRiskDetector(BaseDetector):
    """Detects risky configuration changes in infrastructure and deployment files"""

    def __init__(self, repo_path: str):
        super().__init__(repo_path)

        # Configuration file patterns
        self.config_patterns = [
            r'.*\.ya?ml$',
            r'.*\.json$',
            r'.*\.tf$',
            r'.*\.tfvars$',
            r'Dockerfile',
            r'docker-compose.*\.ya?ml$',
            r'.*\.env$',
            r'.*\.conf$',
            r'.*\.ini$',
        ]

        # Kubernetes risky changes
        self.k8s_risks = {
            'privileged': 0.9,
            'hostNetwork': 0.8,
            'hostPID': 0.8,
            'hostIPC': 0.7,
            'runAsRoot': 0.6,
            'allowPrivilegeEscalation': 0.7,
            'readOnlyRootFilesystem.*false': 0.5,
            'capabilities.*add.*SYS_ADMIN': 0.9,
            'securityContext.*privileged.*true': 0.9,
        }

        # Terraform risky patterns
        self.terraform_risks = {
            'public_access': [
                r'cidr_blocks.*=.*\["0\.0\.0\.0/0"\]',
                r'source_cidr_blocks.*=.*\["0\.0\.0\.0/0"\]',
                r'ingress.*from_port.*=.*0.*to_port.*=.*65535',
            ],
            'insecure_protocols': [
                r'protocol.*=.*"http"',
                r'ssl_policy.*=.*"ELBSecurityPolicy-2016-08"',
                r'min_tls_version.*=.*"1\.0"',
            ],
            'weak_encryption': [
                r'kms_key_id.*=.*""',
                r'encryption.*=.*false',
                r'encrypted.*=.*false',
            ]
        }

        # Docker security issues
        self.docker_risks = [
            (r'FROM.*:latest', 'latest_tag', 0.3),
            (r'USER.*root', 'root_user', 0.6),
            (r'COPY.*\*', 'wildcard_copy', 0.4),
            (r'RUN.*curl.*\|.*sh', 'pipe_to_shell', 0.8),
            (r'--no-check-certificate', 'no_cert_check', 0.7),
            (r'chmod.*777', 'permissive_permissions', 0.5),
        ]

    async def analyze(self, context: ChangeContext) -> DetectorResult:
        """Analyze configuration risks in changes"""
        config_risks = []
        total_risk_score = 0.0
        reasons = []
        anchors = []
        evidence = {
            "k8s_security_issues": [],
            "terraform_exposures": [],
            "docker_vulnerabilities": [],
            "resource_limit_changes": [],
            "network_exposures": []
        }

        for file_change in context.files_changed:
            if not self._is_config_file(file_change.path):
                continue

            # Analyze configuration file
            file_risk = await self._analyze_config_file(file_change)

            if file_risk['score'] > 0:
                config_risks.append(file_risk)
                total_risk_score += file_risk['score']
                reasons.extend(file_risk['reasons'])
                anchors.extend(file_risk['anchors'])

                # Merge evidence
                for key in evidence:
                    if key in file_risk:
                        evidence[key].extend(file_risk[key])

        # Normalize score
        if config_risks:
            risk_score = min(total_risk_score / len(config_risks), 1.0)

            # Auto-escalate for critical security configurations
            critical_issues = [
                item for item in evidence.get('k8s_security_issues', [])
                if item.get('severity') == 'critical'
            ]
            if critical_issues:
                risk_score = max(risk_score, 0.9)
        else:
            risk_score = 0.0

        return DetectorResult(
            score=risk_score,
            reasons=reasons[:5],
            anchors=anchors[:10],
            evidence=evidence,
            execution_time_ms=0.0
        )

    async def _analyze_config_file(self, file_change: FileChange) -> Dict[str, Any]:
        """Analyze a specific configuration file"""
        file_risk = {
            'score': 0.0,
            'reasons': [],
            'anchors': [],
            'k8s_security_issues': [],
            'terraform_exposures': [],
            'docker_vulnerabilities': [],
            'resource_limit_changes': [],
            'network_exposures': []
        }

        try:
            file_path = Path(self.repo_path) / file_change.path

            if not file_path.exists():
                return file_risk

            content = file_path.read_text(encoding='utf-8')

            # Determine file type and analyze accordingly
            if self._is_kubernetes_file(file_change.path, content):
                await self._analyze_kubernetes_config(content, file_change, file_risk)
            elif self._is_terraform_file(file_change.path):
                await self._analyze_terraform_config(content, file_change, file_risk)
            elif self._is_docker_file(file_change.path):
                await self._analyze_docker_config(content, file_change, file_risk)
            else:
                await self._analyze_generic_config(content, file_change, file_risk)

        except Exception as e:
            file_risk['reasons'].append(f"Failed to analyze {file_change.path}: {str(e)}")

        return file_risk

    async def _analyze_kubernetes_config(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Analyze Kubernetes configuration for security issues"""
        lines = content.split('\n')

        for line_num, line in enumerate(lines, 1):
            line_stripped = line.strip().lower()

            # Check for security context issues
            for risk_pattern, risk_weight in self.k8s_risks.items():
                if re.search(risk_pattern.lower(), line_stripped):
                    severity = 'critical' if risk_weight >= 0.8 else 'high' if risk_weight >= 0.6 else 'medium'

                    file_risk['k8s_security_issues'].append({
                        'line': line_num,
                        'issue': risk_pattern,
                        'content': line.strip(),
                        'severity': severity,
                        'risk_weight': risk_weight
                    })

                    file_risk['score'] += risk_weight
                    file_risk['reasons'].append(f"K8s security issue: {risk_pattern} at line {line_num}")
                    file_risk['anchors'].append(f"{file_change.path}:{line_num}")

            # Check for missing resource limits
            if 'container' in line_stripped and 'image:' in line_stripped:
                # Look for resource limits in next few lines
                has_limits = any('limits:' in lines[i].lower() for i in range(line_num, min(line_num + 10, len(lines))))

                if not has_limits:
                    file_risk['resource_limit_changes'].append({
                        'line': line_num,
                        'issue': 'missing_resource_limits',
                        'severity': 'medium'
                    })
                    file_risk['score'] += 0.3
                    file_risk['reasons'].append(f"Missing resource limits at line {line_num}")

        # Check for network policies and service exposures
        await self._check_k8s_network_exposure(content, file_change, file_risk)

    async def _analyze_terraform_config(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Analyze Terraform configuration for security issues"""
        lines = content.split('\n')

        for line_num, line in enumerate(lines, 1):
            # Check for public access patterns
            for pattern in self.terraform_risks['public_access']:
                if re.search(pattern, line, re.IGNORECASE):
                    file_risk['terraform_exposures'].append({
                        'line': line_num,
                        'type': 'public_access',
                        'pattern': pattern,
                        'content': line.strip(),
                        'severity': 'high'
                    })
                    file_risk['score'] += 0.7
                    file_risk['reasons'].append(f"Public access exposure at line {line_num}")

            # Check for insecure protocols
            for pattern in self.terraform_risks['insecure_protocols']:
                if re.search(pattern, line, re.IGNORECASE):
                    file_risk['score'] += 0.5
                    file_risk['reasons'].append(f"Insecure protocol at line {line_num}")

            # Check for weak encryption
            for pattern in self.terraform_risks['weak_encryption']:
                if re.search(pattern, line, re.IGNORECASE):
                    file_risk['score'] += 0.6
                    file_risk['reasons'].append(f"Weak encryption setting at line {line_num}")

    async def _analyze_docker_config(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Analyze Dockerfile for security issues"""
        lines = content.split('\n')

        for line_num, line in enumerate(lines, 1):
            for pattern, issue_type, risk_weight in self.docker_risks:
                if re.search(pattern, line, re.IGNORECASE):
                    severity = 'critical' if risk_weight >= 0.8 else 'high' if risk_weight >= 0.6 else 'medium'

                    file_risk['docker_vulnerabilities'].append({
                        'line': line_num,
                        'type': issue_type,
                        'pattern': pattern,
                        'content': line.strip(),
                        'severity': severity
                    })

                    file_risk['score'] += risk_weight
                    file_risk['reasons'].append(f"Docker security issue: {issue_type} at line {line_num}")
                    file_risk['anchors'].append(f"{file_change.path}:{line_num}")

    async def _analyze_generic_config(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Analyze generic configuration files"""
        lines = content.split('\n')

        generic_risks = [
            (r'debug\s*=\s*true', 'debug_enabled', 0.3),
            (r'ssl\s*=\s*false', 'ssl_disabled', 0.7),
            (r'auth.*=\s*false', 'auth_disabled', 0.8),
            (r'verify\s*=\s*false', 'verification_disabled', 0.6),
            (r'password\s*=\s*["\']?\w+["\']?', 'hardcoded_password', 0.9),
            (r'secret\s*=\s*["\']?\w+["\']?', 'hardcoded_secret', 0.9),
        ]

        for line_num, line in enumerate(lines, 1):
            for pattern, issue_type, risk_weight in generic_risks:
                if re.search(pattern, line, re.IGNORECASE):
                    file_risk['score'] += risk_weight
                    file_risk['reasons'].append(f"Config issue: {issue_type} at line {line_num}")
                    file_risk['anchors'].append(f"{file_change.path}:{line_num}")

    async def _check_k8s_network_exposure(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Check for Kubernetes network exposure issues"""
        try:
            # Try to parse as YAML
            docs = yaml.safe_load_all(content)

            for doc in docs:
                if not doc or not isinstance(doc, dict):
                    continue

                kind = doc.get('kind', '').lower()

                if kind == 'service':
                    service_type = doc.get('spec', {}).get('type', '').lower()
                    if service_type in ['nodeport', 'loadbalancer']:
                        file_risk['network_exposures'].append({
                            'type': 'service_exposure',
                            'service_type': service_type,
                            'severity': 'medium'
                        })
                        file_risk['score'] += 0.4
                        file_risk['reasons'].append(f"Service exposed via {service_type}")

                elif kind == 'ingress':
                    # Check for wildcard hosts or missing TLS
                    spec = doc.get('spec', {})
                    rules = spec.get('rules', [])

                    for rule in rules:
                        host = rule.get('host', '')
                        if '*' in host:
                            file_risk['network_exposures'].append({
                                'type': 'wildcard_ingress',
                                'host': host,
                                'severity': 'medium'
                            })
                            file_risk['score'] += 0.3

                    if not spec.get('tls'):
                        file_risk['score'] += 0.2
                        file_risk['reasons'].append("Ingress without TLS configuration")

        except yaml.YAMLError:
            # Not valid YAML, skip structured analysis
            pass

    def _is_config_file(self, file_path: str) -> bool:
        """Check if file is a configuration file"""
        return any(re.search(pattern, file_path, re.IGNORECASE) for pattern in self.config_patterns)

    def _is_kubernetes_file(self, file_path: str, content: str) -> bool:
        """Check if file is a Kubernetes configuration"""
        if not file_path.endswith(('.yml', '.yaml')):
            return False

        k8s_indicators = ['apiVersion:', 'kind:', 'metadata:', 'spec:']
        return any(indicator in content for indicator in k8s_indicators)

    def _is_terraform_file(self, file_path: str) -> bool:
        """Check if file is a Terraform configuration"""
        return file_path.endswith(('.tf', '.tfvars'))

    def _is_docker_file(self, file_path: str) -> bool:
        """Check if file is a Dockerfile"""
        return 'dockerfile' in file_path.lower() or file_path.endswith('.dockerfile')