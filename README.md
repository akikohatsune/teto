<p align="center">
  <img src="teto.jpg" alt="Kasane Teto" width="500">
</p>
<p align="center">
  <span style="color:#8a8f98; font-style: italic;">"39! NVIDIA Edition - The Chimera of Lost Music is here."</span>
</p>

<h1 align="center">Teto (NVIDIA NIM)</h1>

<p align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/NVIDIA-76B900?style=for-the-badge&logo=nvidia&logoColor=white" alt="NVIDIA">
  <img src="https://img.shields.io/badge/Discord.go-5865F2?style=for-the-badge&logo=discord&logoColor=white" alt="Discord">
  <img src="https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge" alt="License">
</p>

**Teto** là một Discord Bot AI hiệu năng cao, được xây dựng bằng ngôn ngữ **Go** và tối ưu hóa thông qua **NVIDIA NIM**. Dự án tập trung vào tốc độ phản hồi cực nhanh, tính cá nhân hóa sâu sắc và hệ thống bảo mật đa lớp. 

Dựa trên nhân vật **Kasane Teto**, bot mang phong cách tinh nghịch, năng động và đôi khi hơi "tsundere", mang lại trải nghiệm trò chuyện sống động hơn là một trợ lý ảo khô khan.

---

## ✨ Tính năng nổi bật (Key Features)

- ⚡ **NVIDIA NIM Integration:** Sử dụng model `google/gemma-3n-e4b-it` (mặc định) cho tốc độ xử lý vượt trội và khả năng phân tích hình ảnh chính xác.
- 🧠 **User-Based Memory Isolation:** Mỗi người dùng có một "bộ não" riêng biệt. Lịch sử trò chuyện được cô lập hoàn toàn theo `user_id`, đảm bảo tính riêng tư và không bị nhầm lẫn ngữ cảnh.
- 🛡️ **KomiFilter 2.0:** Hệ thống bảo mật nâng cao giúp chặn đứng Prompt Injection, Jailbreak và đặc biệt là chống rò rỉ thông tin hệ thống (Prompt/Response Leak).
- 🔄 **Visual Env Sync:** Tự động đồng bộ và kiểm tra file cấu hình `.env` khi khởi động, đảm bảo bot luôn đủ điều kiện vận hành.
- 📉 **Tối ưu tài nguyên:** Được viết bằng Go, bot cực kỳ nhẹ, tiêu tốn ít RAM và xử lý đồng thời (concurrency) tốt.

---

## 🚀 Cài đặt nhanh (Quick Start)

### Yêu cầu hệ thống
- [Go](https://go.dev/dl/) (phiên bản 1.20 trở lên)
- Một API Key từ [NVIDIA API Catalog](https://build.nvidia.com/)
- Một Discord Bot Token từ [Discord Developer Portal](https://discord.com/developers/applications)

### Các bước thực hiện
1. **Clone repository:**
   ```bash
   git clone https://github.com/akikohatsune/teto.git
   cd teto
   ```

2. **Cài đặt dependencies:**
   ```bash
   go mod tidy
   ```

3. **Cấu hình:**
   Hệ thống sẽ tự động tạo file `.env` từ `.env.example` khi chạy lần đầu. Bạn cần mở `.env` và điền các thông tin:
   - `NVIDIA_API_KEY`: Key từ NVIDIA.
   - `DISCORD_TOKEN`: Token của bot.
   - `OWNER_USER_ID`: ID Discord của bạn (để dùng lệnh Admin).

4. **Chạy bot:**
   ```bash
   go run main.go
   ```

---

## 🛠️ Lệnh điều khiển (Commands)

### 💬 Trò chuyện & AI
- **@Teto + [Nội dung]:** Chat trực tiếp với Teto.
- **`!chat <nội dung>`:** Chat bằng lệnh (hỗ trợ phân tích hình ảnh đính kèm).
- **`!ask <nội dung>`:** Bí danh của lệnh chat.

### 🔐 Quản lý (Admin Only)
- **`!clearmemo`:** Xóa sạch bộ nhớ ngắn hạn của **tất cả người dùng**.
- **`!ban @user [lý do]`:** Cấm người dùng sử dụng bot.
- **`!removeban @user`:** Gỡ lệnh cấm.
- **`!replayTeto ls`:** Xem danh sách nhật ký chat gần đây.
- **`!replayTeto <id>`:** Xem chi tiết nội dung chat theo ID.

### ⚙️ Hệ thống
- **`!provider`:** Hiển thị thông tin model và cấu hình AI hiện tại.
- **`!terminated on|off`:** Bật/tắt chế độ bảo trì (tạm dừng phản hồi).

---

## 🛡️ Bảo mật với KomiFilter

`KomiFilter` là trái tim bảo mật của Teto, hoạt động qua 3 giai đoạn:
1. **User Injection Shield:** Phát hiện và chặn các kỹ thuật thao túng prompt từ người dùng.
2. **System Prompt Protection:** Ngăn chặn các yêu cầu yêu cầu tiết lộ mã nguồn hoặc "system rules".
3. **Smart Response Scrubbing:** Tự động kiểm tra và ẩn phản hồi từ AI nếu nó vô tình tiết lộ các thông tin nhạy cảm của hệ thống.

---

## 📂 Cấu trúc lưu trữ

Teto sử dụng SQLite để quản lý dữ liệu một cách bền vững và độc lập:
- `chat_memory.db`: Lưu trữ lịch sử hội thoại (đã mã hóa/cô lập theo User).
- `ban_control.db`: Danh sách đen người dùng.
- `callnames.db`: Lưu trữ biệt danh cá nhân hóa mà người dùng đặt cho Teto hoặc ngược lại.

---

## 📜 Giấy phép & Tín dụng

- **License:** MIT
- **Character Design:** Kasane Teto (Official Diva)
- **Artwork by:** [gomya0_0](https://x.com/gomya0_0)
- **Developed by:** [Akiko Hatsune](https://github.com/akikohatsune)

---
<p align="center"><i>"Don't let my music fade away... and don't forget to check the .env file!"</i></p>
