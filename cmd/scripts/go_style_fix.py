#!/usr/bin/env python3
"""Fix Go code style: add package comments and blank lines between functions."""
import re, os

PKG_COMMENTS = {
    'cmd/server/main.go': 'provides the nts-gater server entry point.',
    'internal/api/router.go': 'provides HTTP API handlers for the web frontend.',
    'internal/config/config.go': 'provides configuration loading and management.',
    'internal/database/database.go': 'provides MySQL database initialization and connection management.',
    'internal/dispatcher/dispatcher.go': 'provides message dispatching to registered handlers.',
    'internal/errors/codes.go': 'provides business error code definitions.',
    'internal/generator/generator.go': 'provides test data generation utilities for charging scenarios.',
    'internal/model/model.go': 'provides database models for test reports and message archives.',
    'internal/handlers/auth.go': 'provides authentication, key update and basic business handlers.',
    'internal/handlers/auth_test.go': 'provides tests for authentication handlers.',
    'internal/handlers/charging.go': 'provides charging business handlers.',
    'internal/handlers/misc.go': 'provides miscellaneous business handlers (BMS, billing, config, upgrade).',
    'internal/recorder/recorder.go': 'provides per-session message recording and statistics.',
    'internal/report/pdf.go': 'provides PDF test report generation.',
    'internal/report/service.go': 'provides test report persistence and query services.',
    'internal/scenario/basic_charging.go': 'provides the basic charging test scenario.',
    'internal/scenario/sftp_upgrade.go': 'provides the SFTP upgrade test scenario.',
    'internal/scenario/scenario.go': 'provides the test scenario engine and Scenario interface.',
    'internal/scenario/errors.go': 'provides error definitions for the scenario engine.',
    'internal/server/tcp.go': 'provides TCP server for charging station connections.',
    'internal/session/manager.go': 'provides charging station session management.',
    'internal/validator/frame.go': 'provides frame and message validation.',
    'internal/protocol/codec/checksum.go': 'provides checksum calculation for protocol frames.',
    'internal/protocol/codec/frame.go': 'provides frame encoding and decoding.',
    'internal/protocol/codec/scanner.go': 'provides TCP stream scanning and frame extraction.',
    'internal/protocol/crypto/aes_cbc.go': 'provides AES-256-CBC encryption and decryption.',
    'internal/protocol/crypto/aes_cbc_test.go': 'provides tests for AES-256-CBC encryption.',
    'internal/protocol/standard/enums.go': 'provides standard protocol enum definitions.',
    'internal/protocol/standard/protocol.go': 'provides the standard protocol implementation.',
    'internal/protocol/standard/register.go': 'provides message registration for the standard protocol.',
    'internal/protocol/standard/registry.go': 'provides the message registry implementation.',
    'internal/protocol/types/enums.go': 'provides protocol type constants and direction/funcCode name mappings.',
    'internal/protocol/types/message.go': 'provides core message and message spec interfaces.',
    'internal/protocol/types/protocol.go': 'provides the Protocol interface and related type definitions.',
    'internal/protocol/standard/msg/auth.go': 'provides 0x0A/0x0B authentication message definitions.',
    'internal/protocol/standard/msg/basic.go': 'provides 0x01 heartbeat and 0x23 time sync message definitions.',
    'internal/protocol/standard/msg/billing.go': 'provides 0x22 billing rules message definitions.',
    'internal/protocol/standard/msg/bms.go': 'provides BMS-related message definitions (0x10, 0x24-0x27).',
    'internal/protocol/standard/msg/charging.go': 'provides charging message definitions (0x03-0x08).',
    'internal/protocol/standard/msg/codec.go': 'provides BCD/ASCII/BYTE field read/write utilities.',
    'internal/protocol/standard/msg/codec_test.go': 'provides tests for field codec utilities.',
    'internal/protocol/standard/msg/config.go': 'provides 0x0C/0xC1/0xC2 configuration message definitions.',
    'internal/protocol/standard/msg/errors.go': 'provides error definitions for message codec.',
    'internal/protocol/standard/msg/reservation.go': 'provides 0x07 reservation and 0x08 platform stop message definitions.',
    'internal/protocol/standard/msg/upgrade.go': 'provides 0xF6/0xF7 SFTP upgrade message definitions.',
    'test/codec/codec_test.go': 'provides integration tests for frame encoding/decoding.',
}

def fix_file(filepath):
    rel = os.path.relpath(filepath, '/Users/xiongyang/CodeBuddy/gater')
    with open(filepath, 'r') as f:
        content = f.read()

    lines = content.split('\n')
    new_lines = []
    pkg_name = None
    has_pkg_comment = False

    for i, line in enumerate(lines):
        stripped = line.strip()
        m = re.match(r'^package\s+(\w+)', stripped)
        if m:
            pkg_name = m.group(1)
            # Check if there's already a comment above the package line
            for j in range(len(new_lines)-1, -1, -1):
                pj = new_lines[j].strip()
                if pj == '':
                    break
                if pj.startswith('//') or pj.startswith('/*') or pj == '*' or pj == '*/':
                    has_pkg_comment = True
                break
            
            if not has_pkg_comment and rel in PKG_COMMENTS:
                comment = f"// Package {pkg_name} {PKG_COMMENTS[rel]}"
                new_lines.append(comment)
        
        new_lines.append(line)

    result = '\n'.join(new_lines)

    # Add blank line between closing brace and next func/type/var/const/comment
    result = re.sub(r'\}\n(func |type |var |const )', r'}\n\n\1', result)
    result = re.sub(r'\}\n(//)', r'}\n\n\1', result)
    # Collapse multiple blank lines
    result = re.sub(r'\}\n\n\n+', r'}\n\n', result)
    # Also between // === section headers and func
    # Already handled by the comment rule above

    with open(filepath, 'w') as f:
        f.write(result)

for root, dirs, files in os.walk('/Users/xiongyang/CodeBuddy/gater'):
    dirs[:] = [d for d in dirs if d not in ('vendor', '.git', 'docs', 'node_modules')]
    for f in sorted(files):
        if f.endswith('.go'):
            fix_file(os.path.join(root, f))

print("Done!")
