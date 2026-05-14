# Security Policy

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security vulnerability in Trust-Tunnel, please report it responsibly.

**Do NOT open a public GitHub issue for security vulnerabilities.**

Instead, please report security issues by emailing the maintainers directly:

- lj388371@antgroup.com

Please include the following information:

- Type of vulnerability (e.g., buffer overflow, injection, privilege escalation)
- Full paths of source file(s) related to the vulnerability
- The location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Impact of the issue, including how an attacker might exploit it

## Response Timeline

- We will acknowledge receipt of your vulnerability report within 48 hours
- We will provide a more detailed response within 7 days indicating the next steps
- We will keep you informed of the progress towards a fix
- We may ask for additional information or guidance

## Security Best Practices

When deploying Trust-Tunnel:

- Always use TLS/NTLS encryption for communication between Client and Agent
- Restrict access to the Agent port using network policies
- Run the Agent container with least-privilege principles
- Regularly update the sidecar image to the latest version
- Enable audit logging for all sessions
- Configure the authentication plugin to control access