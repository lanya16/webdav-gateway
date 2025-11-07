#!/bin/bash

# WebDAV Gateway 项目 Git 配置脚本
# 版本: v1.0.0
# 最后更新: 2025-11-06

set -e

echo "🔧 WebDAV Gateway Git 配置脚本"
echo "================================"
echo ""

# 检查Git是否安装
if ! command -v git &> /dev/null; then
    echo "❌ Git 未安装，请先安装 Git"
    echo "📥 安装指南:"
    echo "   - Ubuntu/Debian: sudo apt-get install git"
    echo "   - CentOS/RHEL: sudo yum install git"
    echo "   - macOS: brew install git"
    echo "   - Windows: https://git-scm.com/download/win"
    exit 1
fi

# 显示Git版本
git_version=$(git --version)
echo "✅ $git_version"
echo ""

# 配置Git用户信息
echo "👤 配置 Git 用户信息"
echo "请提供以下信息："

read -p "您的姓名: " user_name
if [ -z "$user_name" ]; then
    echo "❌ 用户名不能为空"
    exit 1
fi

read -p "您的邮箱: " user_email
if [ -z "$user_email" ]; then
    echo "❌ 邮箱不能为空"
    exit 1
fi

# 设置Git配置
echo ""
echo "🔧 设置 Git 配置..."
git config user.name "$user_name"
git config user.email "$user_email"

# 设置提交模板
if [ -f ".gitmessage" ]; then
    git config commit.template ".gitmessage"
    echo "✅ 已设置提交消息模板"
else
    echo "⚠️  .gitmessage 文件不存在，跳过模板设置"
fi

# 设置编辑器（可选）
echo ""
read -p "是否设置 Vim 作为默认编辑器？(y/N): " set_editor
if [[ $set_editor =~ ^[Yy]$ ]]; then
    git config core.editor "vim"
    echo "✅ 已设置 Vim 作为默认编辑器"
fi

# 配置Git别名
echo ""
echo "⚡ 配置 Git 别名..."

# 基础别名
git config alias.st status
git config alias.co checkout
git config alias.br branch
git config alias.ci commit
git config alias.lg "log --color --graph --pretty=format:'%Cred%h%Creset -%C(yellow)%d%Creset %s %Cgreen(%cr) %C(bold blue)<%an>%Creset' --abbrev-commit"

# 提交相关别名
git config alias.c "commit -m"
git config alias.ca "commit --amend"
git config alias.cw "commit --wip"

# 分支相关别名
git config alias.b "branch"
git config alias.ba "branch -a"
git config alias.bd "branch -d"
git config alias.bD "branch -D"

# 差异查看别名
git config alias.d "diff"
git config alias.ds "diff --staged"
git config alias.dh "diff HEAD"

# 日志查看别名
git config alias.l "log --oneline --graph"
git config alias.ll "log --oneline"
git config alias.la "log --all"

# 搜索相关别名
git config alias.f "fetch"
git config alias.s "show"
git config alias.r "remote -v"

echo "✅ 已配置 20 个 Git 别名"

# 配置钩子
echo ""
echo "🪝 配置 Git 钩子..."

if [ -d ".git/hooks" ]; then
    # 预提交钩子
    if [ -f ".git/hooks/pre-commit" ]; then
        echo "✅ pre-commit 钩子已存在"
    else
        echo "📝 需要手动创建 pre-commit 钩子"
    fi
    
    # 提交后钩子
    if [ -f ".git/hooks/post-commit" ]; then
        echo "✅ post-commit 钩子已存在"
    else
        echo "📝 需要手动创建 post-commit 钩子"
    fi
else
    echo "⚠️  .git/hooks 目录不存在，可能不是 Git 仓库"
fi

# 验证配置
echo ""
echo "🔍 验证配置结果..."

echo "📋 当前 Git 配置："
echo "   用户名: $(git config user.name)"
echo "   邮箱: $(git config user.email)"
echo "   编辑器: $(git config core.editor || echo '默认编辑器')"
echo "   提交模板: $(git config commit.template || echo '未设置')"

echo ""
echo "⚡ 可用的别名："
echo "   git st        -> git status"
echo "   git co        -> git checkout"
echo "   git br        -> git branch"
echo "   git ci        -> git commit"
echo "   git c         -> git commit -m"
echo "   git lg        -> 彩色日志"
echo "   git d         -> git diff"
echo "   git l         -> 简化日志"
echo ""
echo "📋 测试用法示例："
echo "   git st                    # 查看状态"
echo "   git c \"feat: 提交测试\"     # 快速提交"
echo "   git lg                    # 查看历史"
echo ""

# 显示提交消息示例
echo "📝 提交消息格式示例："
echo "   feat(lock): 实现LOCK/UNLOCK核心功能"
echo "   fix(handler): 修复请求解析错误"
echo "   test(unit): 添加单元测试"
echo "   docs(readme): 更新README文档"
echo ""

# 显示文件状态
if [ -f ".gitignore" ]; then
    echo "✅ .gitignore 文件已存在"
else
    echo "⚠️  .gitignore 文件不存在，请检查项目配置"
fi

if [ -f "CHANGELOG.md" ]; then
    echo "✅ CHANGELOG.md 文件已存在"
else
    echo "⚠️  CHANGELOG.md 文件不存在，请检查项目配置"
fi

# 检查文档
if [ -f "docs/GIT_COMMIT_GUIDELINES.md" ]; then
    echo "✅ Git 提交规范文档已存在"
else
    echo "⚠️  Git 提交规范文档不存在"
fi

# 总结和建议
echo ""
echo "🎉 Git 配置完成！"
echo ""
echo "📚 后续步骤："
echo "   1. 查看提交规范: cat docs/GIT_COMMIT_GUIDELINES.md"
echo "   2. 查看提交模板: cat .gitmessage"
echo "   3. 测试提交功能: git commit -m \"test(config): 验证git配置\""
echo "   4. 查看历史记录: git lg"
echo ""
echo "📖 相关文档："
echo "   - docs/GIT_COMMIT_GUIDELINES.md - 详细提交规范"
echo "   - .gitmessage - 提交消息模板"
echo "   - CHANGELOG.md - 版本更新日志"
echo "   - TODO_待办列表_完整版.md - 项目待办事项"
echo ""
echo "💡 使用提示："
echo "   - 使用 'git status' 检查工作区状态"
echo "   - 使用 'git lg' 查看清晰的提交历史"
echo "   - 遵循 Conventional Commits 格式提交"
echo "   - 定期更新 CHANGELOG.md"
echo ""

# 检查是否有未跟踪的修改
if git status --porcelain | grep -q .; then
    echo "⚠️  工作区有未跟踪或修改的文件"
    echo "   建议在提交前先检查状态：git status"
else
    echo "✅ 工作区干净，没有未跟踪的修改"
fi

echo ""
echo "🚀 准备好开始开发了！"