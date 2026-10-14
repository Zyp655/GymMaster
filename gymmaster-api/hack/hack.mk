# Lệnh của GoFrame CLI (cài: go install github.com/gogf/gf/cmd/gf/v2@v2.9.0)
# Lưu ý: oh-my-zsh có alias `gf` = git fetch; dùng `command gf` hoặc `unalias gf`.

ctrl: ## Sinh lại interface api/<module>/<module>.go + stub controller từ Req/Res
	command gf gen ctrl

.PHONY: ctrl
