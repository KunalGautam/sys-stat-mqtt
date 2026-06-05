# Sys-Stat-MQTT Daemon

A lightweight, platform-agnostic Go daemon that monitors system statistics (CPU usage, RAM usage, Disk utilization, CPU/HDD temperatures, and system load averages) and publishes them as a clean, rounded JSON payload to an MQTT broker at configurable intervals.

## Features

- **Resource Utilization**: CPU, memory, and disk usage percentages.
- **Hardware Temperatures**: Tracks CPU and HDD/SSD temperatures (robustly auto-detects composite NVMe and core temps).
- **System Load**: Reports 1, 5, and 15-minute system load averages.
- **Reliability**: Supports auto-reconnection, KeepAlive settings, and QoS 1 to guarantee delivery.
- **Friendly Logging**: Prints human-readable logs of publishes to the terminal/syslog.
- **Clean Telemetry**: All decimal values are cleanly rounded to 2 decimal places.
- **Service Ready**: Designed to run cleanly as a system service.

---

## JSON Payload Schema

Every publish sends a JSON object resembling the following:

```json
{
  "cpu_percent": 2.63,
  "memory_percent": 25.2,
  "disk_percent": 21.44,
  "cpu_temp_c": 58.00,
  "disk_temp_c": 47.85,
  "load_1m": 0.67,
  "load_5m": 0.72,
  "load_15m": 0.74,
  "details": {
    "cpu_model": "11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz",
    "cpu_logical_cores": 8,
    "cpu_physical_cores": 4,
    "total_memory_gb": 15.4,
    "total_disk_gb": 474.92
  }
}
```

---

## Configuration (`.env`)

The daemon reads configuration from environment variables or a `.env` file in its working directory.

An example `.env` file:

```env
MQTT_BROKER=tcp://localhost:1883
MQTT_CLIENT_ID=mac-linux-sysmon
MQTT_TOPIC=system/stats
STATS_INTERVAL=5s
MQTT_USERNAME=my_username
MQTT_PASSWORD=my_password
```

| Key | Default Value | Description |
| :--- | :--- | :--- |
| `MQTT_BROKER` | `tcp://localhost:1883` | The MQTT broker URI (supports `tcp://` and `ssl://`). |
| `MQTT_CLIENT_ID` | `mac-linux-sysmon` | Unique client identifier. |
| `MQTT_TOPIC` | `system/stats` | Base MQTT topic. The Client ID is appended (`system/stats<ClientID>`) for dynamic topics. |
| `STATS_INTERVAL` | `5s` | Stat reporting frequency (e.g., `5s`, `30s`, `1m`, `1h`). |
| `MQTT_USERNAME` | *(Optional)* | Username if the broker requires authorization. |
| `MQTT_PASSWORD` | *(Optional)* | Password if the broker requires authorization. |

---

## Building and Testing

To run tests:
```bash
go test -v ./...
```

To build locally:
```bash
go build -o sys-stat-mqtt main.go
```

To build for all architectures/platforms (runs cross-compilation and outputs to `dist/`):
```bash
./build.sh
```

---

## Running as a Service/Daemon

### 1. Linux (using `systemd`)

1. **Move Binary & Config**:
   Copy your architecture-specific binary to `/usr/local/bin` and create a config directory:
   ```bash
   sudo cp dist/sys-stat-mqtt-linux-amd64 /usr/local/bin/sys-stat-mqtt
   sudo chmod +x /usr/local/bin/sys-stat-mqtt
   sudo mkdir -p /etc/sys-stat-mqtt
   sudo cp .env /etc/sys-stat-mqtt/.env
   sudo chmod 600 /etc/sys-stat-mqtt/.env
   ```

2. **Create Service File**:
   Create a systemd unit file at `/etc/systemd/system/sys-stat-mqtt.service`:
   ```ini
   [Unit]
   Description=System Statistics MQTT Publisher Daemon
   After=network.target

   [Service]
   Type=simple
   User=root
   WorkingDirectory=/etc/sys-stat-mqtt
   EnvironmentFile=/etc/sys-stat-mqtt/.env
   ExecStart=/usr/local/bin/sys-stat-mqtt
   Restart=on-failure
   RestartSec=5s

   [Install]
   WantedBy=multi-user.target
   ```

3. **Start and Enable Service**:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable sys-stat-mqtt
   sudo systemctl start sys-stat-mqtt
   ```

4. **View Logs**:
   ```bash
   journalctl -u sys-stat-mqtt -f
   ```

---

### 2. macOS (using `launchd`)

1. **Move Binary & Config**:
   ```bash
   sudo cp dist/sys-stat-mqtt-darwin-amd64 /usr/local/bin/sys-stat-mqtt # OR darwin-arm64 for Apple Silicon
   sudo chmod +x /usr/local/bin/sys-stat-mqtt
   sudo mkdir -p /Library/Application\ Support/sys-stat-mqtt
   sudo cp .env /Library/Application\ Support/sys-stat-mqtt/.env
   sudo chmod 600 /Library/Application\ Support/sys-stat-mqtt/.env
   ```

2. **Create plist Configuration**:
   Create a daemon property list at `/Library/LaunchDaemons/com.abhashtech.sys-stat-mqtt.plist`:
   ```xml
   <?xml version="1.0" encoding="UTF-8"?>
   <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
   <plist version="1.0">
   <dict>
       <key>Label</key>
       <string>com.abhashtech.sys-stat-mqtt</string>
       <key>ProgramArguments</key>
       <array>
           <string>/usr/local/bin/sys-stat-mqtt</string>
       </array>
       <key>WorkingDirectory</key>
       <string>/Library/Application Support/sys-stat-mqtt</string>
       <key>RunAtLoad</key>
       <true/>
       <key>KeepAlive</key>
       <true/>
       <key>StandardOutPath</key>
       <string>/var/log/sys-stat-mqtt.log</string>
       <key>StandardErrorPath</key>
       <string>/var/log/sys-stat-mqtt.err</string>
   </dict>
   </plist>
   ```

3. **Load and Run Daemon**:
   ```bash
   sudo launchctl bootstrap system /Library/LaunchDaemons/com.abhashtech.sys-stat-mqtt.plist
   ```

4. **Verify / View Logs**:
   ```bash
   tail -f /var/log/sys-stat-mqtt.log
   ```

---

### 3. Windows (using NSSM)

Because the binary does not natively implement the Windows Service API, it is best run using **NSSM (Non-Sucking Service Manager)**.

1. **Prepare Binary and Configuration**:
   - Choose a directory (e.g., `C:\Program Files\sys-stat-mqtt`).
   - Copy `dist/sys-stat-mqtt-windows-amd64.exe` to that directory and rename it to `sys-stat-mqtt.exe`.
   - Copy your `.env` file into the same directory (`C:\Program Files\sys-stat-mqtt\.env`).

2. **Install with NSSM**:
   - Download NSSM from [nssm.cc](https://nssm.cc/) and place the `nssm.exe` in your Path.
   - Open a command prompt as **Administrator** and run:
     ```cmd
     nssm install sys-stat-mqtt
     ```
   - An NSSM UI will pop up:
     - **Application Tab**:
       - *Path*: `C:\Program Files\sys-stat-mqtt\sys-stat-mqtt.exe`
       - *Startup directory*: `C:\Program Files\sys-stat-mqtt`
     - **Details Tab**:
       - *Display name*: `System Stats MQTT Daemon`
       - *Description*: `Publishes system resource specs and metrics to MQTT.`
       - *Startup type*: `Automatic`
     - **I/O Tab**:
       - Redirect Output (stdout) and Error (stderr) to `C:\Program Files\sys-stat-mqtt\sys-stat-mqtt.log`.
     - Click **Install service**.

3. **Start the Service**:
   You can start it through the Windows Services management console (`services.msc`), or via command prompt:
   ```cmd
   nssm start sys-stat-mqtt
   ```
