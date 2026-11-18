# Lazypodman

**The lazier way to manage everything podman.**

Lazypodman là terminal UI (TUI) để quản lý Podman: container, image, volume,
network, project compose, logs, stats và các thao tác thường dùng — tất cả trong
một cửa sổ, mỗi thao tác chỉ cách một phím bấm.

Đây là bản fork của [lazydocker](https://github.com/jesseduffield/lazydocker)
(MIT, Jesse Duffield) được chuyển sang nói chuyện với **Podman service API**
thay vì Docker daemon. Toàn bộ giao diện, panel và keybinding đến từ lazydocker;
phần khác biệt nằm ở cách dò endpoint Podman, compose và các chi tiết Podman
(xem [Khác biệt so với lazydocker](#khác-biệt-so-với-lazydocker)).

**Mục lục**

- [Yêu cầu](#yêu-cầu)
- [Cài đặt](#cài-đặt)
- [Lazypodman tìm Podman service ở đâu](#lazypodman-tìm-podman-service-ở-đâu)
- [Hướng dẫn sử dụng](#hướng-dẫn-sử-dụng)
- [Cấu hình](#cấu-hình)
- [Khác biệt so với lazydocker](#khác-biệt-so-với-lazydocker)
- [Xử lý sự cố](#xử-lý-sự-cố)
- [Phát triển & kiểm thử](#phát-triển--kiểm-thử)
- [Giấy phép](#giấy-phép)

---

## Yêu cầu

| Thành phần | Phiên bản / ghi chú |
| --- | --- |
| Podman | >= **4.0**, có service API (socket) đang chạy |
| Compose (tuỳ chọn) | `podman compose` (Podman >= 4.1) hoặc script `podman-compose` |
| Go (chỉ khi build) | >= **1.22** |
| Hệ điều hành | Linux, macOS (`podman machine`), Windows (`podman machine`) |

Lazypodman dùng **Docker-compatible API** do Podman cung cấp, nên cần bật
socket/service trước khi chạy:

```sh
# Rootless (phổ biến nhất)
systemctl --user enable --now podman.socket

# Rootful
sudo systemctl enable --now podman.socket
```

Kiểm tra nhanh:

```sh
podman info                      # daemon/service đang chạy?
echo $XDG_RUNTIME_DIR/podman/podman.sock
```

Trên macOS/Windows cần khởi động máy ảo Podman:

```sh
podman machine start
podman system connection list    # xem URI để dùng cho CONTAINER_HOST
```

---

## Cài đặt

### 1. Build từ source (khuyến nghị)

Dependencies đã được vendor sẵn, không cần mạng sau khi clone:

```sh
git clone https://github.com/lazypodman/lazypodman.git
cd lazypodman
GOFLAGS=-mod=vendor go build -o lazypodman .
./lazypodman
```

Hoặc cài vào `$GOBIN`/`$GOPATH/bin`:

```sh
GOFLAGS=-mod=vendor go install -mod=vendor
```

Chạy thử không cần build:

```sh
GOFLAGS=-mod=vendor go run . --help
```

### 2. Binary release

Tải file nén cho nền tảng của bạn ở trang Releases, giải nén và đặt
`lazypodman` vào một thư mục trong `PATH`:

```sh
curl -L -o lazypodman.tar.gz \
  https://github.com/lazypodman/lazypodman/releases/latest/download/lazypodman_<version>_Linux_x86_64.tar.gz
tar xzvf lazypodman.tar.gz lazypodman
install -Dm 755 lazypodman -t "$HOME/.local/bin"
```

### 3. Chạy trong container Podman

```sh
podman build -t lazypodman .

podman run --rm -it \
  --security-opt label=disable \
  -v /run/podman/podman.sock:/run/podman/podman.sock \
  -e CONTAINER_HOST=unix:///run/podman/podman.sock \
  -v ./config:/root/.config/lazypodman \
  lazypodman
```

Repo cũng có `docker-compose.yml` (dùng được với `podman-compose` hoặc
`podman compose`) cho cùng thiết lập trên:

```sh
podman-compose run --rm lazypodman   # hoặc: podman compose run --rm lazypodman
```

### 4. Alias cho nhanh

```sh
echo "alias lzp='lazypodman'" >> ~/.zshrc   # hoặc ~/.bashrc
```

---

## Lazypodman tìm Podman service ở đâu

Theo thứ tự ưu tiên giảm dần:

| # | Nguồn | Ví dụ |
| --- | --- | --- |
| 1 | `PODMAN_HOST` (riêng của lazypodman) | `unix:///run/user/1000/podman/podman.sock` |
| 2 | `CONTAINER_HOST` (biến chuẩn của Podman CLI) | `tcp://127.0.0.1:8888` |
| 3 | `DOCKER_HOST` (vẫn tôn trọng, cho podman-docker/CI) | `unix:///run/podman/podman.sock` |
| 4 | Socket rootless đang tồn tại | `$XDG_RUNTIME_DIR/podman/podman.sock`, `/run/user/<uid>/podman/podman.sock` |
| 5 | Socket rootful đang tồn tại | `/run/podman/podman.sock` |
| 6 | Default của nền tảng | Windows: `npipe:////./pipe/podman-machine-default` |

- Đường dẫn trần (không có scheme) cũng được chấp nhận:
  `CONTAINER_HOST=/run/user/1000/podman/podman.sock` → tự thêm `unix://`.
- Endpoint `ssh://` được tunnel tự động về socket local:
  ```sh
  CONTAINER_HOST=ssh://user@host/run/user/1000/podman/podman.sock lazypodman
  ```
  (không ghi path thì mặc định tunnel tới `/run/podman/podman.sock` trên host)
- Docker context (`~/.docker/contexts`) **không** được dùng — hãy dùng biến môi
  trường ở trên.

---

## Hướng dẫn sử dụng

Chạy trong thư mục có file compose để thấy thêm panel Services/Project:

```sh
cd /duong/dan/co/compose-file
lazypodman
```

### Cờ dòng lệnh

| Cờ | Ý nghĩa |
| --- | --- |
| `-f, --file` | Chỉ định compose file khác (lặp lại được nhiều lần) |
| `-p, --project` | Đặt tên compose project |
| `-c, --config` | In config mặc định (sau khi merge với config của bạn) |
| `-d, --debug` | Bật log debug |
| `--version` / `-h` | Phiên bản / trợ giúp |

### Bố cục màn hình

- Cột trái: **Project → Services → Containers → Images → Volumes → Networks**
  (panel Project/Services chỉ hiện khi bạn đang ở trong thư mục compose hoặc
  truyền `-p`).
- Panel phải (main): các tab **Logs / Stats / Config / Env / Top / Config
  compose / About** tuỳ item đang chọn.

### Điều hướng chung

| Phím | Tác dụng |
| --- | --- |
| `1`…`6` | Nhảy tới panel Projects/Services/Containers/Images/Volumes/Networks |
| `tab` / `shift+tab` | Panel kế tiếp / trước đó (hoặc `l` / `h`) |
| `↑` `↓` / `k` `j` | Di chuyển trong danh sách |
| `enter` | Focus panel chính (xem chi tiết item) |
| `[` / `]` | Tab trước / kế tiếp trong panel chính |
| `/` | Lọc danh sách |
| `esc` | Quay lại / đóng panel phụ |
| `q` | Thoát (khi không có panel phụ) |
| `+` / `_` | Đổi screen mode (normal / half / fullscreen) |
| `o` / `e` | Mở / sửa file config (ở panel Project) |

### Phím theo panel

**Containers**

| Phím | Tác dụng |
| --- | --- |
| `d` | Xoá container (`podman rm`) |
| `e` | Ẩn/hiện container đã dừng |
| `p` | Pause / unpause |
| `s` | Stop |
| `r` | Restart |
| `a` | Attach vào container |
| `m` | Xem logs (tab Logs) |
| `E` | Mở shell trong container (`podman exec -it …`) |
| `c` | Chạy custom command |
| `b` | Bulk commands (thao tác hàng loạt) |
| `w` | Mở port đầu tiên (http) bằng browser |

**Services**

| Phím | Tác dụng |
| --- | --- |
| `u` / `U` | `compose up` service / cả project |
| `D` | `compose down` project |
| `s` / `S` | Stop / start service |
| `r` / `R` | Restart / menu restart |
| `p` | Pause |
| `a` | Attach |
| `m` | Logs |
| `E` | Exec shell |
| `c` / `b` | Custom command / bulk command |
| `w` | Mở port bằng browser |

**Images / Volumes / Networks**

| Phím | Tác dụng |
| --- | --- |
| `d` | Xoá (image/volume/network) |
| `c` / `b` | Custom command / bulk command (bao gồm prune) |

> Danh sách đầy đủ cho mọi ngôn ngữ: [docs/keybindings](/docs/keybindings).
> Sinh lại bằng `GOFLAGS=-mod=vendor go run ./scripts/cheatsheet generate`.

### Quy trình thường gặp

1. **Debug một service lỗi**: chọn service → `m` xem logs, `]` sang tab Stats
   để xem CPU/RAM, `E` để vào shell.
2. **Sửa code xong cần deploy lại**: chọn service → `R` → chọn recreate/rebuild
   (dùng `commandTemplates.recreateService` / `rebuildService`).
3. **Dọn dẹp ổ đĩa**: panel Images / Volumes / Networks → `b` → prune.
4. **Xem compose config thực tế**: chọn project → tab **Compose Config**.

---

## Cấu hình

### Vị trí file config

| OS | Đường dẫn |
| --- | --- |
| Linux | `~/.config/lazypodman/config.yml` |
| macOS | `~/Library/Application Support/jesseduffield/lazypodman/config.yml` (hoặc `~/Library/Application Support/lazypodman/config.yml` trên máy mới) |
| Windows | `C:\Users\<User>\AppData\Roaming\lazypodman\config.yml` |

- Ghi đè thư mục: biến môi trường `CONFIG_DIR`.
- Mở nhanh trong app: panel **Project** → `o` (mở) hoặc `e` (sửa).
- Xem config sau khi merge với default: `lazypodman -c`.
- Thay đổi chỉ có hiệu lực sau khi khởi động lại lazypodman.

Tham chiếu đầy đủ mọi field: [docs/Config.md](/docs/Config.md) và
[godoc pkg/config](https://pkg.go.dev/github.com/lazypodman/lazypodman/pkg/config).

### Các nhóm cấu hình chính

```yaml
gui:
  language: "auto"              # auto | en | pl | nl | de | tr | zh | fr | es | pt
  border: "rounded"             # rounded | single | double | hidden
  theme:
    activeBorderColor: [green, bold]
    inactiveBorderColor: [default]
    selectedLineBgColor: [blue]
    optionsTextColor: [blue]
  sidePanelWidth: 0.3333        # tỉ lệ chiều rộng panel trái
  showBottomLine: true
  wrapMainPanel: true
  ignoreMouseEvents: false
  returnImmediately: false      # không dừng ở màn hình "press enter to return"
  screenMode: "normal"          # normal | half | fullscreen
  containerStatusHealthStyle: "long"  # long | short | icon
  expandFocusedSidePanel: true
confirmOnQuit: false
```

```yaml
logs:
  timestamps: false
  since: '60m'      # '' = lấy toàn bộ logs
  tail: ''          # ví dụ 200 = chỉ lấy 200 dòng cuối

commandTemplates:
  podman: podman                 # binary podman dùng cho attach/exec (đổi thành podman-remote nếu remote)
  podmanCompose: podman compose  # hoặc "podman-compose", "podman compose -f a.yml -f b.yml"
  restartService: '{{ .PodmanCompose }} restart {{ .Service.Name }}'
  up: '{{ .PodmanCompose }} up -d'
  down: '{{ .PodmanCompose }} down'
  downWithVolumes: '{{ .PodmanCompose }} down --volumes'
  upService: '{{ .PodmanCompose }} up -d {{ .Service.Name }}'
  startService: '{{ .PodmanCompose }} start {{ .Service.Name }}'
  stopService: '{{ .PodmanCompose }} stop {{ .Service.Name }}'
  serviceLogs: '{{ .PodmanCompose }} logs --since=60m --follow {{ .Service.Name }}'
  viewServiceLogs: '{{ .PodmanCompose }} logs --follow {{ .Service.Name }}'
  rebuildService: '{{ .PodmanCompose }} up -d --build {{ .Service.Name }}'
  recreateService: '{{ .PodmanCompose }} up -d --force-recreate {{ .Service.Name }}'
  allLogs: '{{ .PodmanCompose }} logs --tail=300 --follow'
  viewAlLogs: '{{ .PodmanCompose }} logs'
  composeConfig: '{{ .PodmanCompose }} config'
  checkComposeConfig: '{{ .PodmanCompose }} config --quiet'
  serviceTop: '{{ .PodmanCompose }} top {{ .Service.Name }}'
```

```yaml
stats:
  maxDuration: 3m
  graphs:
    - caption: CPU (%)
      statPath: DerivedStats.CPUPercentage
      color: cyan
    - caption: Memory (%)
      statPath: DerivedStats.MemoryPercentage
      color: green
```

### Custom commands / Bulk commands

Dùng được Go template với `{{ .Podman }}`, `{{ .PodmanCompose }}`,
`{{ .Container }}`, `{{ .Service }}`, `{{ .Image }}`, `{{ .Volume }}`,
`{{ .Network }}`, `{{ .Project }}`:

```yaml
customCommands:
  containers:
    - name: bash
      attach: true
      command: '{{ .Podman }} exec -it {{ .Container.ID }} bash'
    - name: logs 100 dòng
      attach: true
      shell: true
      command: '{{ .Podman }} logs --tail 100 {{ .Container.ID }} | less'

bulkCommands:
  containers:
    - name: prune stopped
      command: 'podman container prune -f'
  images:
    - name: prune dangling
      command: 'podman image prune -f'
```

`attach: true` → chuyển sang subprocess tương tác; `attach: false` → chạy nền.
`serviceNames` giới hạn command cho một số service nhất định.

### Biến môi trường

| Biến | Ý nghĩa |
| --- | --- |
| `PODMAN_HOST` | Endpoint Podman (ưu tiên cao nhất) |
| `CONTAINER_HOST` | Endpoint Podman (chuẩn của podman CLI) |
| `DOCKER_HOST` | Vẫn được đọc nếu hai biến trên trống |
| `CONFIG_DIR` | Ghi đè thư mục chứa `config.yml` |
| `DEBUG=TRUE` | Bật log debug (tương đương `-d`) |

---

## Khác biệt so với lazydocker

- **Chỉ Podman**: mặc định dò socket rootless/rootful và biến của Podman,
  không dùng Docker context store.
- **Ẩn container infra**: mỗi pod có container `-infra` (pause) ẩn; lazypodman
  lọc khỏi panel Containers giống `podman ps`.
- **Compose linh hoạt**: thử `podman compose` trước, tự fallback
  `podman-compose`; đọc cả label `com.docker.compose.*` và
  `io.podman.compose.*`.
- **Binary cấu hình được**: `commandTemplates.podman` cho phép dùng
  `podman-remote` hoặc đường dẫn tuyệt đối.
- **Tên config/khoá**: `commandTemplates.podmanCompose` (thay cho
  `dockerCompose`), `commandTemplates.composeConfig`.
- **Stats an toàn hơn**: tránh chia 0/Inf khi Podman không trả
  `system_cpu_usage` hoặc memory limit = 0.

---

## Xử lý sự cố

**"Can't reach the podman service socket"**
→ Service chưa chạy hoặc socket khác vị trí. Bật `podman.socket` (xem
[Yêu cầu](#yêu-cầu)), hoặc xuất `CONTAINER_HOST` trỏ đúng socket.

**Lỗi "connection to the podman service failed"**
→ Kiểm tra `podman info` chạy được bằng chính user đang dùng. Với rootless,
chạy đúng user (không dùng `sudo`) và đảm bảo `XDG_RUNTIME_DIR` được set.

**Panel Services trống**
→ Bạn không ở trong thư mục compose, thiếu `podman compose`/`podman-compose`,
hoặc compose file không hợp lệ. Thử `podman compose config --quiet` (hoặc
`podman-compose config --quiet`) ngay trong thư mục đó.

**Không thấy logs / số liệu**
→ Mặc định chỉ lấy logs 60 phút gần nhất (`logs.since`). Xem thêm
[FAQ logs](/docs/Config.md).

**Không select được text bằng chuột**
→ Giữ `option`/`alt` khi kéo chuột, hoặc tắt chuột bằng
`gui.ignoreMouseEvents: true`.

---

## Phát triển & kiểm thử

```sh
export PATH=$PATH:/usr/local/go/bin   # nếu Go chưa có trong PATH

GOFLAGS=-mod=vendor go build ./...
GOFLAGS=-mod=vendor go test ./...
GOFLAGS=-mod=vendor go vet ./...
gofmt -l main.go pkg/ scripts/

# Sinh lại docs/keybindings sau khi sửa i18n
GOFLAGS=-mod=vendor go run ./scripts/cheatsheet generate
```

Xem thêm [CONTRIBUTING.md](CONTRIBUTING.md) và [CLAUDE.md](CLAUDE.md).

---

## Giấy phép

Lazypodman là bản fork của **lazydocker**
(https://github.com/jesseduffield/lazydocker), phát hành theo **MIT License**.

Bản quyền của dự án gốc và toàn bộ mã nguồn kế thừa được giữ nguyên:

```
MIT License

Copyright (c) 2018 Jesse Duffield

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- Mã nguồn gốc: lazydocker — Copyright (c) 2018 Jesse Duffield, MIT.
- Các thay đổi cho Podman trong repo này cũng phát hành theo MIT; xem
  [LICENSE](LICENSE) để biết nội dung đầy đủ.
