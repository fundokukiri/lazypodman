# Removal lock (danh sách cấm xóa cho container/image)

## Intent

Một lần bấm nhầm `d` (hoặc bulk remove all) xóa mất container/image đang cần.
Người dùng đánh dấu trước những mục "cấm xóa"; khi xóa mục đó phải gõ 6 ký tự
đầu của ID thì mới xóa. Mục không nằm trong danh sách thì xóa như bình thường.
Không đụng gì đến stop/restart/kill/pause/exec/logs.

## Goals

- Bật/tắt khóa cho mục đang chọn bằng một phím trong panel containers và images.
- Danh sách khóa lưu lại giữa các lần chạy.
- Xóa mục đã khóa: prompt hỏi 6 ký tự đầu của ID; sai thì không xóa gì.
- Mục đã khóa hiện dấu `[locked] ` trước tên trong panel.
- Lô có nhiều mục khóa (marked containers) → hỏi lần lượt từng mục.

## Non-goals

- Không phải ranh giới bảo mật: file danh sách là text thường, user sửa được.
- Không khóa volume/network/pod, không chặn prune (xem Known limits).
- Không khóa custom command của user.

## Approaches

1. **Danh sách ID + prompt ở tầng GUI** (chọn). Hai helper trong `pkg/gui`
   (`containerRemovalLocks` / `imageRemovalLocks` + `confirmLockedRemovals`)
   bọc callback xóa ở 3 handler: image remove menu, container remove menu,
   bulk remove all. Không đụng `pkg/commands`.
2. So mật khẩu chung trong config: đã bàn và bị thay bằng yêu cầu mới (gõ ID).
3. Khóa từng item bằng flag trong `pkg/commands`: không hỏi được từ đó, và
   ảnh hưởng cả custom command.

## Design

- `pkg/config/removal_lock.go`:
  - `RemovalLock{Containers []string, Images []string}` (ID đầy đủ).
  - `(*AppConfig) LoadRemovalLock()` / `SaveRemovalLock()` đọc/ghi
    `<ConfigDir>/removal-lock.yml` (cạnh `config.yml`, quyền 0600).
  - `(*RemovalLock) ContainerLocked/ImageLocked/ToggleContainer/ToggleImage`.
- `pkg/gui/removal_lock.go`:
  - `shortID(id)` = 6 ký tự đầu sau khi bỏ tiền tố `sha256:`.
  - `confirmLockedRemovals(locks []lockedRemoval, after func() error) error`:
    rỗng → chạy `after()`; ngược lại mở prompt (title = `Tr.RemovalLockPrompt`),
    so khớp không phân biệt hoa thường; sai → error panel; đúng → đệ quy mục
    tiếp theo, hết thì `after()`.
  - `handleToggleContainerLock` / `handleToggleImageLock`: đảo trạng thái, ghi
    file, flash status, rerender list.
- Keybindings: `L` ở panel containers và images.
- Panel: thêm tiền tố `[locked] ` vào ô tên (containers) / ô name (images).
- i18n: 5 chuỗi tiếng Anh mới trong `englishSet`; ngôn ngữ khác fallback EN.
- Test: `pkg/config/removal_lock_test.go` (load/save/toggle), trong `pkg/gui`:
  `shortID`, lọc locked, `confirmLockedRemovals` với danh sách rỗng chạy
  callback.

## Known limits

- Prune (`D` / bulk "prune") vẫn xóa mục đã khóa, vì prune chạy qua podman CLI
  và không đi qua danh sách chọn. Ghi rõ trong `docs/Config.md` và comment code.
- Danh sách khóa theo ID: container bị xóa rồi tạo lại bằng cùng tên sẽ có ID
  mới, tức là không còn khóa.
