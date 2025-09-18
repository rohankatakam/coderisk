"""
Security Risk Detector

Mini-SAST detector for common security vulnerabilities including:
- SQL injection patterns
- Path traversal vulnerabilities
- Unsafe YAML/XML parsing
- Hard-coded secrets
- Command injection
- XSS vulnerabilities

Time budget: 30-120ms
"""

import re
import hashlib
import math
from typing import Dict, List, Set, Optional, Any
from pathlib import Path

from . import BaseDetector, DetectorResult, ChangeContext, FileChange, detector_registry


@detector_registry.register
class SecurityRiskDetector(BaseDetector):
    """Detects security vulnerabilities and anti-patterns in code changes"""

    def __init__(self, repo_path: str):
        super().__init__(repo_path)

        # Security vulnerability patterns
        self.vulnerability_patterns = {
            'sql_injection': [
                r'query\s*=.*\+.*["\']',  # String concatenation in queries
                r'execute\(.*%.*\)',  # String formatting in queries
                r'cursor\.execute\(.*\+',  # Concatenated SQL
                r'SELECT.*\+.*FROM',  # SQL concatenation
                r'WHERE.*\+.*=',  # WHERE clause concatenation
            ],
            'command_injection': [
                r'os\.system\(.*\+',  # os.system with concatenation
                r'subprocess\.(call|run|Popen)\(.*\+',  # subprocess with concatenation
                r'exec\(.*input\(',  # exec with user input
                r'eval\(.*request\.',  # eval with request data
                r'shell=True.*\+',  # shell=True with concatenation
            ],
            'path_traversal': [
                r'open\(.*\+.*\)',  # File operations with concatenation
                r'file\(.*\+.*\)',  # File access with concatenation
                r'\.\./',  # Directory traversal patterns
                r'os\.path\.join\(.*request\.',  # Path join with user input
                r'pathlib\.Path\(.*input\(',  # Path creation with input
            ],
            'xss': [
                r'innerHTML\s*=.*\+',  # innerHTML with concatenation
                r'document\.write\(.*\+',  # document.write with concatenation
                r'\.html\(.*\+',  # jQuery html() with concatenation
                r'render_template_string\(',  # Flask template string rendering
                r'HttpResponse\(.*\+',  # Django HttpResponse with concatenation
            ],
            'unsafe_deserialization': [
                r'pickle\.loads?\(',  # Python pickle
                r'yaml\.load\(',  # Unsafe YAML loading
                r'eval\(',  # Eval function
                r'JSON\.parse\(.*request\.',  # JSON.parse with request data
                r'unserialize\(',  # PHP unserialize
            ],
            'crypto_issues': [
                r'MD5\(',  # Weak hash function
                r'SHA1\(',  # Weak hash function
                r'DES\(',  # Weak encryption
                r'ECB',  # Weak cipher mode
                r'Random\(\)',  # Weak random number generation
            ]
        }

        # Secret patterns with entropy thresholds
        self.secret_patterns = [
            (r'["\']([A-Za-z0-9+/]{40,}={0,2})["\']', 'base64_encoded'),  # Base64
            (r'["\']([A-Fa-f0-9]{32,})["\']', 'hex_encoded'),  # Hex strings
            (r'password\s*=\s*["\']([^"\']{8,})["\']', 'password'),  # Passwords
            (r'api[_-]?key\s*=\s*["\']([^"\']{10,})["\']', 'api_key'),  # API keys
            (r'secret\s*=\s*["\']([^"\']{10,})["\']', 'secret'),  # Secrets
            (r'token\s*=\s*["\']([^"\']{10,})["\']', 'token'),  # Tokens
        ]

        # High entropy threshold for secret detection
        self.entropy_threshold = 3.5

    async def analyze(self, context: ChangeContext) -> DetectorResult:
        """Analyze security risks in code changes"""
        security_risks = []
        total_risk_score = 0.0
        reasons = []
        anchors = []
        evidence = {
            "vulnerabilities": [],
            "secrets_detected": [],
            "unsafe_patterns": [],
            "crypto_issues": []
        }

        for file_change in context.files_changed:
            if file_change.change_type == 'deleted':
                continue

            file_path = Path(self.repo_path) / file_change.path

            if not file_path.exists() or not self._is_analyzable_file(file_path):
                continue

            # Analyze security issues in this file
            file_risk = await self._analyze_file_security(file_change, file_path)

            if file_risk['score'] > 0:
                security_risks.append(file_risk)
                total_risk_score += file_risk['score']
                reasons.extend(file_risk['reasons'])
                anchors.extend(file_risk['anchors'])

                # Merge evidence
                for key in evidence:
                    if key in file_risk:
                        evidence[key].extend(file_risk[key])

        # Normalize score
        if security_risks:
            risk_score = min(total_risk_score / len(context.files_changed), 1.0)

            # Auto-escalate for critical security issues
            for vuln in evidence.get('vulnerabilities', []):
                if vuln.get('severity') == 'critical':
                    risk_score = max(risk_score, 0.9)

            for secret in evidence.get('secrets_detected', []):
                if secret.get('confidence') == 'high':
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

    async def _analyze_file_security(self, file_change: FileChange, file_path: Path) -> Dict[str, Any]:
        """Analyze security issues in a specific file"""
        file_risk = {
            'score': 0.0,
            'reasons': [],
            'anchors': [],
            'vulnerabilities': [],
            'secrets_detected': [],
            'unsafe_patterns': [],
            'crypto_issues': []
        }

        try:
            content = file_path.read_text(encoding='utf-8')

            # Check for vulnerability patterns
            await self._check_vulnerability_patterns(content, file_change, file_risk)

            # Check for secrets and credentials
            await self._check_secrets(content, file_change, file_risk)

            # Check configuration files for security issues
            if self._is_config_file(file_path):
                await self._check_config_security(content, file_change, file_risk)

        except Exception as e:
            file_risk['reasons'].append(f"Failed to analyze {file_path.name}: {str(e)}")

        return file_risk

    async def _check_vulnerability_patterns(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Check for common vulnerability patterns"""
        lines = content.split('\n')

        for vuln_type, patterns in self.vulnerability_patterns.items():
            for pattern in patterns:
                matches = re.finditer(pattern, content, re.IGNORECASE | re.MULTILINE)

                for match in matches:
                    line_num = content[:match.start()].count('\n') + 1
                    line_content = lines[line_num - 1] if line_num <= len(lines) else ""

                    severity = self._assess_vulnerability_severity(vuln_type, match.group())
                    risk_weight = self._get_vulnerability_risk_weight(vuln_type, severity)

                    file_risk['vulnerabilities'].append({
                        'type': vuln_type,
                        'line': line_num,
                        'pattern': match.group(),
                        'code': line_content.strip(),
                        'severity': severity
                    })

                    file_risk['score'] += risk_weight
                    file_risk['reasons'].append(f"{vuln_type.replace('_', ' ').title()} vulnerability at line {line_num}")
                    file_risk['anchors'].append(f"{file_change.path}:{line_num}")

    async def _check_secrets(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Check for hard-coded secrets and credentials"""
        lines = content.split('\n')

        for line_num, line in enumerate(lines, 1):
            # Skip comments and empty lines
            if re.match(r'^\s*(#|//|/\*|\*)', line) or not line.strip():
                continue

            for pattern, secret_type in self.secret_patterns:
                matches = re.finditer(pattern, line, re.IGNORECASE)

                for match in matches:
                    if len(match.groups()) > 0:
                        potential_secret = match.group(1)

                        # Calculate entropy to reduce false positives
                        entropy = self._calculate_entropy(potential_secret)
                        confidence = self._assess_secret_confidence(potential_secret, secret_type, entropy)

                        if confidence in ['medium', 'high'] or entropy > self.entropy_threshold:
                            severity = 'critical' if confidence == 'high' else 'high'
                            risk_weight = 0.9 if severity == 'critical' else 0.6

                            file_risk['secrets_detected'].append({
                                'type': secret_type,
                                'line': line_num,
                                'entropy': round(entropy, 2),
                                'confidence': confidence,
                                'severity': severity,
                                'length': len(potential_secret)
                            })

                            file_risk['score'] += risk_weight
                            file_risk['reasons'].append(f"Potential {secret_type} at line {line_num}")
                            file_risk['anchors'].append(f"{file_change.path}:{line_num}")

    async def _check_config_security(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Check configuration files for security issues"""
        lines = content.split('\n')

        config_security_patterns = [
            (r'debug\s*=\s*true', 'debug_enabled', 'medium'),
            (r'ssl\s*=\s*false', 'ssl_disabled', 'high'),
            (r'verify\s*=\s*false', 'verification_disabled', 'high'),
            (r'auth\s*=\s*false', 'auth_disabled', 'critical'),
            (r'cors\s*=\s*\*', 'cors_wildcard', 'medium'),
            (r'password\s*=\s*["\']?\w+["\']?', 'hardcoded_password', 'critical'),
        ]

        for line_num, line in enumerate(lines, 1):
            for pattern, issue_type, severity in config_security_patterns:
                if re.search(pattern, line, re.IGNORECASE):
                    risk_weight = 0.9 if severity == 'critical' else 0.6 if severity == 'high' else 0.3

                    file_risk['unsafe_patterns'].append({
                        'type': issue_type,
                        'line': line_num,
                        'pattern': line.strip(),
                        'severity': severity
                    })

                    file_risk['score'] += risk_weight
                    file_risk['reasons'].append(f"Unsafe configuration: {issue_type} at line {line_num}")
                    file_risk['anchors'].append(f"{file_change.path}:{line_num}")

    def _calculate_entropy(self, string: str) -> float:
        """Calculate Shannon entropy of a string"""
        if not string:
            return 0.0

        # Count character frequencies
        frequencies = {}
        for char in string:
            frequencies[char] = frequencies.get(char, 0) + 1

        # Calculate entropy
        entropy = 0.0
        length = len(string)

        for count in frequencies.values():
            probability = count / length
            if probability > 0:
                entropy -= probability * math.log2(probability)

        return entropy

    def _assess_secret_confidence(self, secret: str, secret_type: str, entropy: float) -> str:
        """Assess confidence level for secret detection"""
        # High confidence criteria
        if entropy > 4.5 and len(secret) > 20:
            return 'high'

        # Medium confidence criteria
        if (entropy > self.entropy_threshold and len(secret) > 15) or \
           (secret_type in ['api_key', 'token', 'password'] and len(secret) > 10):
            return 'medium'

        # Low confidence
        if entropy > 2.0 or len(secret) > 8:
            return 'low'

        return 'very_low'

    def _assess_vulnerability_severity(self, vuln_type: str, pattern: str) -> str:
        """Assess the severity of a detected vulnerability"""
        critical_patterns = [
            'exec', 'eval', 'shell=True', 'os.system',
            'pickle.loads', 'yaml.load'
        ]

        high_patterns = [
            'subprocess', 'innerHTML', 'render_template_string'
        ]

        if any(crit in pattern.lower() for crit in critical_patterns):
            return 'critical'
        elif any(high in pattern.lower() for high in high_patterns):
            return 'high'
        else:
            return 'medium'

    def _get_vulnerability_risk_weight(self, vuln_type: str, severity: str) -> float:
        """Get risk weight for vulnerability type and severity"""
        base_weights = {
            'sql_injection': 0.7,
            'command_injection': 0.9,
            'path_traversal': 0.6,
            'xss': 0.5,
            'unsafe_deserialization': 0.8,
            'crypto_issues': 0.4
        }

        severity_multipliers = {
            'critical': 1.5,
            'high': 1.2,
            'medium': 1.0,
            'low': 0.7
        }

        base_weight = base_weights.get(vuln_type, 0.3)
        multiplier = severity_multipliers.get(severity, 1.0)

        return min(base_weight * multiplier, 1.0)

    def _is_config_file(self, file_path: Path) -> bool:
        """Check if file is a configuration file"""
        config_extensions = {'.yml', '.yaml', '.json', '.ini', '.cfg', '.conf', '.env', '.properties'}
        config_names = {'dockerfile', 'makefile', '.env', '.gitignore'}

        return (file_path.suffix.lower() in config_extensions or
                file_path.name.lower() in config_names)

    def _is_analyzable_file(self, file_path: Path) -> bool:
        """Check if file can be analyzed for security issues"""
        # Code files
        code_extensions = {'.py', '.js', '.jsx', '.ts', '.tsx', '.java', '.php', '.rb', '.go', '.rs', '.c', '.cpp'}

        # Config files
        config_extensions = {'.yml', '.yaml', '.json', '.xml', '.ini', '.cfg', '.conf', '.env', '.properties'}

        # Script files
        script_extensions = {'.sh', '.bash', '.ps1', '.bat'}

        analyzable_extensions = code_extensions | config_extensions | script_extensions

        return file_path.suffix.lower() in analyzable_extensions