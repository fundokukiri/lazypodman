# Tính năng của lazypodman

Tài liệu này tổng hợp **những gì đã làm xong và dùng được**, kèm công dụng và cách
dùng. Phần "chưa có" ở cuối để bạn biết ranh giới, không nhầm với tính năng đang
thiết kế.

## 1. lazypodman là gì

TUI quản lý container **chỉ dành cho Podman** (không có chế độ chạy Docker daemon).
Bản gốc là lazydocker (MIT), đã fork và chuyển toàn bộ sang Podman.

- Backend chính: Docker-compatible API của Podman qua socket
  (`$XDG_RUNTIME_DIR/podman/podman.sock` cho rootless, `/run/podman/podman.sock` cho
  rootful). Containers, services, images, volumes, networks, stats, logs đều đi qua
  socket này.
- Backend phụ: CLI `podman`, chỉ dùng cho những thứ API không có — compose, attach,
  exec, top, custom command, **pods** và **healthcheck run**.
- Rootless là ưu tiên; rootful vẫn chạy được.

## 2. Chạy thử

```bash
# cần podman + socket đang chạy (rootless)
systemctl --user start podman.socket

# chạy từ source
GOFLAGS=-mod=vendor go run .

# hoặc build
GOFLAGS=-mod=vendor go build -o lazypodman .
./lazypodman
```

Cờ dòng lệnh: `-p/--project <tên>` (chỉ xem một compose project), `-c/--config`
(in cấu hình đã merge), `-d/--debug`, `--version`.

## 3. Giao diện

Cột trái là các panel, panel giữa là nội dung của mục đang chọn, dòng dưới cùng hiển
thị phím tắt và thông báo trạng thái.

```
Project (1) → Services (2) → Pods (3) → Containers (4) → Images (5) → Volumes (6) → Networks (7)
```

Khi **pods không khả dụng**, panel Pods bị ẩn và số thứ tự co lại còn `1`…`6`
(Project, Services, Containers, Images, Volumes, Networks) — không có lỗ hổng số.

Điều hướng chung cho mọi panel:

| Phím | Việc |
| --- | --- |
| `tab` / `shift+tab` hoặc `l` / `h` | panel kế / trước |
| `↑` `↓` hoặc `k` `j` | di chuyển trong danh sách |
| `enter` | vào panel chính (nội dung) |
| `[` / `]` | tab trước / tab sau của panel chính |
| `/` | lọc danh sách |
| `esc` | đóng popup / quay lại |
| `+` / `_` | đổi chế độ màn hình (normal / half / full) |
| `H` `J` `K` `L` | cuộn nội dung panel chính |
| `q` | thoát |
| `?` | trợ giúp phím tắt |

## 4. Bảng tính năng đã có

| Tính năng | Công dụng | Cách dùng |
| --- | --- | --- |
| Xem container | Danh sách container kèm trạng thái, image, CPU/RAM, health, pod | Panel **Containers** (phím `4`, hoặc `3` khi pods tắt) |
| Logs container | Xem log trực tiếp, có tuỳ chọn tail | Chọn container → tab **Logs** (hoặc `m`) |
| Stats container | CPU, RAM, network, I/O theo thời gian | Tab **Stats** |
| Health tab | Trạng thái health, số lần fail, lịch sử probe | Tab **Health** (sau **Stats**) |
| Events tab | Xem `podman events` trực tiếp (stream) | Tab **Events** trong panel Containers |
| Chọn nhiều (multi-select) | Đánh dấu nhiều container rồi stop/restart/pause/remove một lượt | `space` để đánh dấu, rồi `s` `r` `p` `d` trong Containers |
| Command palette | Danh sách hành động để nhảy nhanh, không cần nhớ phím | `ctrl+p` |
| Quadlet export | Xuất container ra file `.container` cho Quadlet | Panel Containers → `Q` (preview rồi xác nhận) |
| Auto-update | Chạy `podman auto-update`, xem output trực tiếp | Panel Containers → `b` → **Auto-update containers** |
| Chạy healthcheck | Chạy `podman healthcheck run` một lần | Panel Containers → `t` |
| Env / Config / Top | Biến môi trường, cấu hình đầy đủ, tiến trình trong container | Các tab **Env**, **Config**, **Top** |
| Vòng đời container | start / stop / restart / pause / remove (có xác nhận) | `s` `r` `p` `d` trong Containers; `S` start trong Services |
| Attach / Exec shell | Vào shell hoặc gắn vào TTY của container | `a` (attach), `E` (exec shell) |
| Shell tab | Gõ lệnh trong container ngay trong TUI, output stream vào tab | Tab **Shell** (sau **Logs**) → `enter` → gõ lệnh |
| Mở port trên browser | Mở port publish đầu tiên (nếu là http) | `w` |
| Ẩn/hiện container đã dừng | Gọn danh sách | `e` |
| Services (compose) | Gom container theo compose project/service, chạy up/down | Panel **Services**: `u` up service, `U` up project, `D` down project, `R` restart menu |
| Pods | Xem/sửa pod Podman, logs gộp, stats tổng, container trong pod, config | Panel **Pods** (`3` khi bật) — chi tiết ở mục 5 |
| Pod column | Cột pod trong danh sách container | Tự bật khi pods khả dụng; tắt bằng `gui.showPodColumn: false` |
| Kube generate | Xuất pod ra file YAML để dùng với Kubernetes / lưu trữ | Panel **Pods** → `g` → **Generate kube YAML** → xem preview → xác nhận lưu |
| Kube play / down | Chạy hoặc dừng pod từ file YAML | Panel **Pods** → `g` → **Play** hoặc **Down** → chọn file → xác nhận (hiện full path) |
| Images | Xem, xoá, prune, custom command | Panel **Images**: `d`, `b` |
| Volumes | Xem, xoá, prune | Panel **Volumes**: `d`, `b` |
| Networks | Xem, xoá, prune | Panel **Networks**: `d`, `b` |
| Project | Logs của compose project, compose config, credits | Panel **Project**: `m`, tab **Compose Config** |
| Custom command | Chạy lệnh tự định nghĩa với biến template | `c` ở mọi panel, cấu hình `customCommands` |
| Bulk command | Chạy lệnh hàng loạt / prune | `b` ở mọi panel, cấu hình `bulkCommands` |
| Filter | Lọc danh sách theo chuỗi | `/` |
| Cấu hình | Sửa file config ngay trong app | Panel **Project**: `e` (edit), `o` (open) |

## 5. Pods (Phase 1, đã xong)

Panel **Pods** nằm giữa Services và Containers vì pod chứa container.

**Điều kiện bật:** có binary `podman`, version ≥ 3.0, và CLI nói chuyện cùng store với
socket đang dùng. Nếu không, panel bị ẩn, góc dưới phải hiện `pods off: <lý do>`.
Có thể ép bằng `gui.pods.enabled: auto | always | never`.

**Phím trong panel Pods:**

| Phím | Việc |
| --- | --- |
| `s` / `S` | stop / start pod |
| `r` | restart pod |
| `p` | pause / unpause pod |
| `d` | menu xoá (thường / `--force`), luôn hỏi xác nhận |
| `b` | bulk commands (có sẵn **Prune stopped pods**) |
| `c` | custom command cho pod (`{{ .Pod.Name }}`, `{{ .Pod.ID }}`) |
| `g` | menu **Kube**: generate YAML từ pod, play hoặc down file YAML |
| `enter` | vào panel chính, `[` `]` đổi tab |

**Tab của pod:**

- **Logs**: log gộp của mọi container trong pod. Nếu podman hỗ trợ `pod logs --names`
  thì dùng một tiến trình; nếu không thì chạy song song từng container (tối đa 8,
  container đang chạy trước), phần bị bỏ ghi rõ trên màn hình.
- **Stats**: tổng CPU/RAM của các container trong pod (có ghi chú "sum of
  containers"). Chi tiết từng container vẫn xem ở panel Containers.
- **Containers**: danh sách container thuộc pod.
- **Config**: `pod inspect`.

Container hạ tầng (`*-infra`, pause) được lọc khỏi panel Containers giống `podman ps`.

### Kube: generate / play / down

Bấm `g` trong panel **Pods** để mở menu Kube:

| Mục | Việc |
| --- | --- |
| **Generate kube YAML for this pod** | Chạy `podman kube generate` cho pod đang chọn, hiện YAML ở panel chính (preview), rồi hỏi lưu. File lưu vào `gui.kubeDir` với tên `<tên-pod>.yaml`, quyền `0600`. Hỏi xác nhận kèm **full path**; nếu file đã tồn tại thì hỏi ghi đè, Cancel thì giữ nguyên file cũ. |
| **Play kube YAML…** | Liệt kê file `*.yaml` / `*.yml` trong `gui.kubeDir` có dòng `kind:` ở cấp cao nhất, chọn file rồi xác nhận (full path) mới chạy `podman kube play`. Output được stream nguyên văn vào panel chính. |
| **Down kube YAML…** | Như trên nhưng chạy `podman kube play --down`; câu xác nhận nói rõ pod sẽ bị stop và xoá. |

Chi tiết quan trọng:

- Không bao giờ tự chạy: luôn qua menu + xác nhận.
- Không truyền `--replace` hay `--build` ở phiên bản này.
- Cú pháp được dò lazy: `podman kube generate|play` trước, fallback
  `podman generate kube` / `podman play kube`. Không hard-code version; nếu cả hai
  không có, góc dưới phải hiện `kube generate off:` / `kube play off:`.
- File YAML chứa biến môi trường dạng plaintext (không có Secrets) — câu xác nhận
  cảnh báo điều này.
- Tác vụ stream có thể huỷ: chuyển tab/panel sẽ kill tiến trình `podman` con.
- Có thể đổi thư mục bằng `gui.kubeDir`, hỗ trợ `~`.


## 6. Healthcheck (Phase 2, batch 1 — đã xong)

**Tab Health** (sau **Stats** trong panel Containers) hiển thị, lấy từ dữ liệu inspect
qua socket nên **không tốn thêm tiến trình**:

- trạng thái (`healthy` / `unhealthy` / `starting`…), số lần fail liên tiếp;
- lịch sử probe mới nhất trước: giờ (`HH:MM:SS`), exit code (0 xanh, 1–2 đỏ, khác vàng),
  output (xuống dòng → `⏎`, tối đa 200 ký tự, rỗng → `(no output)`), tối đa 20 dòng.

**Phím `t`** (panel Containers) chạy `podman healthcheck run` một lần:

- container phải đang chạy và có healthcheck (`--health-cmd`), nếu không chỉ hiện
  thông báo, **không chạy gì**;
- kết quả `healthy` / `unhealthy` hiện trên dòng trạng thái trong ~10 giây, đồng thời
  tab Health tự refresh. **Unhealthy là kết quả, không phải lỗi** — không có popup lỗi;
- lỗi thật (không chạy được, exit code lạ) mở error panel với output của podman nguyên văn.

Lần bấm `t` đầu tiên sẽ probe `podman healthcheck run --help` (một tiến trình, làm
lazy). Thiếu subcommand → cache và hiện `health run off: <lý do>`; timeout → lần sau
thử lại. Exit code `1`/`2` lấy theo tài liệu Podman và **chưa được kiểm chứng trên máy
thật**, mọi mã khác đều bị coi là lỗi.

> Lưu ý: chạy tay sẽ được Podman ghi vào health log, có thể đổi failing streak/status
> và kích hoạt `--health-on-failure` nếu container được cấu hình như vậy.

## 7. Custom command và bulk command

```yaml
customCommands:
  containers:
    - name: bash
      attach: true
      command: '{{ .Podman }} exec -it {{ .Container.ID }} bash'
      serviceNames: []

bulkCommands:
  pods: []
```

Biến template dùng được: `{{ .Podman }}`, `{{ .PodmanCompose }}`, `{{ .Container }}`,
`{{ .Service }}`, `{{ .Pod }}`. Đặt `attach: true` nếu lệnh cần chiếm màn hình.

## 8. Cấu hình liên quan (trong `~/.config/lazypodman/config.yml`)

```yaml
gui:
  showPodColumn: true     # cột pod trong panel Containers (mặc định true)
  pods:
    enabled: "auto"       # auto | always | never
  containerStatusHealthStyle: "long"  # long | short | icon, cách hiện health trong list

commandTemplates:
  podman: podman          # đổi thành podman-remote nếu dùng remote
logs:
  tail: "300"             # hoặc "all"
```

Xem toàn bộ lựa chọn trong [docs/Config.md](docs/Config.md). Biến môi trường
`CONFIG_DIR` cho phép đổi thư mục chứa `config.yml` nếu cần.

## 9. Chọn endpoint Podman

Thứ tự ưu tiên:

1. `PODMAN_HOST`
2. `CONTAINER_HOST`
3. `DOCKER_HOST` (cho `podman-docker` hoặc CI)
4. socket rootless `$XDG_RUNTIME_DIR/podman/podman.sock`
5. socket rootful `/run/podman/podman.sock`
6. mặc định theo hệ điều hành (Windows/macOS dùng `podman machine`)

`fd://`, `ssh://`, `tcp://`, `http(s)://` được chấp nhận; với `ssh://` lazypodman tự mở
SSH tunnel. Store của Docker context không bao giờ bị đọc.

## 10. Compose

- Dùng `podman compose` nếu có, fallback `podman-compose`; quyết định một lần lúc khởi
  động.
- Hiểu cả label `com.docker.compose.*` và `io.podman.compose.*`.
- Panel Services chỉ hiện khi đang trong thư mục compose hoặc truyền `-p`.

## 11. Vài workflow mẫu

1. **Service hỏng**: chọn service → `m` xem log → `]` sang Stats → nếu có healthcheck,
   sang tab Health hoặc bấm `t` để chạy probe ngay.
2. **Pod có vấn đề**: panel Pods → `enter` → tab **Logs** xem log gộp → tab
   **Containers** xem container nào lỗi → `s`/`r` thao tác trên cả pod.
3. **Dọn dẹp**: Images `b` → prune; Volumes `b`; Networks `b`; Pods `b` → prune stopped
   pods. Mọi thao tác xoá đều hỏi xác nhận.
4. **Debug nhanh trong container**: `E` để exec shell, `a` để attach, `w` mở port ra
   browser.

## 12. Giới hạn hiện tại

- Pods và healthcheck-run cần CLI `podman` cùng store với socket. Thiết lập chỉ có
  socket (ví dụ `CONTAINER_HOST=ssh://…`, `podman machine`) vẫn dùng được phần còn lại,
  nhưng sẽ thấy `pods off:` / `health run off:` / `cli off:` ở góc dưới phải.
- Pod stats là tổng của các container, không có CPU/RAM riêng của pod.
- Nếu `pod logs --names` không có, log gộp giới hạn 8 container mỗi lần.
- Health tab hiện những gì inspect trả về (Podman giới hạn số probe), tối đa 20 dòng.
- Exit code healthcheck ngoài `0` chưa được kiểm chứng trên máy thật.
- Kube: không truyền `--replace` / `--build`; output play/down nằm ở panel chính
  (đổi tab/panel để quay lại nội dung bình thường). Play/down cần thư mục
  `gui.kubeDir` tồn tại, còn generate thì tự tạo.
- Kube chỉ có trong panel Pods, nên máy chỉ có socket (không có CLI) không dùng được.
- Tab **Shell** là shell dạng dòng lệnh (không PTY): không chạy `vim`/`htop`/`less`,
  không có tab-completion; cần terminal thật thì dùng `E`.
- Events tab và auto-update cần CLI `podman`; không có CLI thì tab Events ghi
  `cli off:` và không spawn gì, còn auto-update chỉ hiện thông báo.
- Quadlet export ghi file `.container` là điểm bắt đầu dễ đọc, không dịch toàn bộ
  mọi field của inspect; env trong unit là plaintext; app không chạy `systemctl`.
- Multi-select đánh dấu theo ID container; refresh danh sách không làm mất dấu,
  nhưng container bị xoá khỏi danh sách thì mất khỏi lựa chọn.


## 13. Chưa có (đừng tìm)

- Push/pull image, secrets, podman machine, `podman generate systemd` (chỉ có Quadlet
  export dạng file, không gọi `systemctl`).

## 14. Mức kiểm chứng của tài liệu này

- Những gì mô tả ở mục 4–10 là hành vi đã implement trong code và đã có unit test
  (chạy `GOFLAGS=-mod=vendor go test ./...` xanh, build Linux/Windows/macOS).
- Các hành vi phụ thuộc Podman thật (pods, healthcheck, kube, compose, socket) được kiểm thử
  bằng **fixture tổng hợp**; **chưa ai chạy checklist tay trên Podman thật**. Khi bạn
  chạy thử, các mục trong checklist của
  [plan pods](docs/superpowers/plans/2026-11-18-pods.md) và
  [plan healthcheck](docs/superpowers/plans/2026-11-18-healthcheck.md) là thứ cần
  đánh dấu.
