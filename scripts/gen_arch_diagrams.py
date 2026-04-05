#!/usr/bin/env python3
"""
生成 AIOPS 智能运维平台 功能图 + 组件图
主色: #513CC8
"""
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches
from matplotlib.patches import FancyBboxPatch
from matplotlib import font_manager
import os

# ── 字体 ──────────────────────────────────────────────
FONT_PATH = "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc"
FONT_BOLD  = "/usr/share/fonts/opentype/noto/NotoSansCJK-Bold.ttc"
font_manager.fontManager.addfont(FONT_PATH)
font_manager.fontManager.addfont(FONT_BOLD)
plt.rcParams['font.family'] = 'Noto Sans CJK SC'
plt.rcParams['axes.unicode_minus'] = False

# ── 颜色 ──────────────────────────────────────────────
PRIMARY      = '#513CC8'
PRIMARY_LIGHT= '#EEE9FB'
PRIMARY_MID  = '#9B8DE0'
WHITE        = '#FFFFFF'
BG           = '#F8F7FF'
BORDER       = '#C5B8F0'
GRAY_TEXT    = '#6B5FAD'
DARK_TEXT    = '#1a1a2e'
GREEN        = '#2ECC71'
TEAL         = '#1ABC9C'
ORANGE       = '#E67E22'

OUT_DIR = os.path.dirname(os.path.abspath(__file__)) + "/../docs"
os.makedirs(OUT_DIR, exist_ok=True)

# ════════════════════════════════════════════════════════
#  工具函数
# ════════════════════════════════════════════════════════
def rounded_box(ax, x, y, w, h, fc, ec, lw=1.5, radius=0.015, alpha=1.0, ls='-'):
    box = FancyBboxPatch((x, y), w, h,
        boxstyle=f"round,pad=0,rounding_size={radius}",
        facecolor=fc, edgecolor=ec, linewidth=lw, alpha=alpha,
        linestyle=ls, transform=ax.transAxes, clip_on=False)
    ax.add_patch(box)
    return box

def center_text(ax, x, y, text, size=9, color=DARK_TEXT, weight='normal', va='center'):
    ax.text(x, y, text, ha='center', va=va, fontsize=size,
            color=color, fontweight=weight, transform=ax.transAxes)

# ════════════════════════════════════════════════════════
#  图1：功能图
# ════════════════════════════════════════════════════════
def draw_feature_diagram():
    fig, ax = plt.subplots(figsize=(16, 10))
    ax.set_xlim(0, 1); ax.set_ylim(0, 1)
    ax.axis('off')
    fig.patch.set_facecolor(BG)

    # ── 标题 ──────────────────────────────────────────
    rounded_box(ax, 0.02, 0.92, 0.96, 0.07, PRIMARY, PRIMARY, lw=0, radius=0.012)
    center_text(ax, 0.5, 0.955, 'AIOPS 智能运维平台 — 功能架构图',
                size=16, color=WHITE, weight='bold')

    # ════ 定义功能域 ════
    # 每个域: (label, x, y, w, h, color, items_grid)
    # items_grid = [(row_items), ...]  每行若干item
    domains = [
        {
            'label': 'AI 智能对话',
            'sub':   'AI Agent · 多轮推理 · 工具调用',
            'x': 0.03, 'y': 0.63, 'w': 0.44, 'h': 0.27,
            'color': PRIMARY,
            'rows': [
                ['即时对话', '自然语言运维', 'Function Calling'],
                ['多模型切换', '流式输出', 'WebSocket 实时通信'],
            ]
        },
        {
            'label': '云平台管理',
            'sub':   'EasyStack · ZStack · 多云接入',
            'x': 0.03, 'y': 0.33, 'w': 0.44, 'h': 0.27,
            'color': '#7B5FD4',
            'rows': [
                ['云主机管理', '云硬盘管理', '网络管理'],
                ['负载均衡', '监控告警', '配额管理'],
            ]
        },
        {
            'label': '运维自动化',
            'sub':   '工作流 · 定时任务 · 消息队列',
            'x': 0.03, 'y': 0.03, 'w': 0.44, 'h': 0.27,
            'color': '#9B5FCF',
            'rows': [
                ['工作流编排', '定时任务(Cron)', '异步任务调度'],
                ['任务日志', '执行监控', '告警通知'],
            ]
        },
        {
            'label': 'AI 模型配置',
            'sub':   '13家厂商 · API管理 · 连通测试',
            'x': 0.52, 'y': 0.63, 'w': 0.46, 'h': 0.27,
            'color': '#3D5BD4',
            'rows': [
                ['OpenAI / DeepSeek', '通义千问 / 智谱 GLM', '硅基流动 / Kimi'],
                ['文心一言 / 豆包', '混元 / 百川', 'Claude / Gemini'],
            ]
        },
        {
            'label': '平台管理',
            'sub':   '用户 · 权限 · 安全',
            'x': 0.52, 'y': 0.33, 'w': 0.46, 'h': 0.27,
            'color': '#2E6EC4',
            'rows': [
                ['用户管理', 'Admin/User 角色', 'JWT 认证'],
                ['密码强度校验', '操作审计', '多主题切换'],
            ]
        },
        {
            'label': '技能中心',
            'sub':   '能力扩展 · 工具集成',
            'x': 0.52, 'y': 0.03, 'w': 0.46, 'h': 0.27,
            'color': '#1A7AB5',
            'rows': [
                ['EasyStack API 技能', 'ZStack API 技能', '自定义脚本'],
                ['Webhook 集成', '资源查询', '批量操作'],
            ]
        },
    ]

    for d in domains:
        x, y, w, h = d['x'], d['y'], d['w'], d['h']
        color = d['color']
        # 外框
        rounded_box(ax, x, y, w, h, PRIMARY_LIGHT, color, lw=2, radius=0.012)
        # 顶部标签栏
        rounded_box(ax, x, y+h-0.07, w, 0.07, color, color, lw=0, radius=0.012)
        # 标题
        center_text(ax, x+w/2, y+h-0.035, d['label'], size=11, color=WHITE, weight='bold')
        # 副标题
        center_text(ax, x+w/2, y+h-0.055, d['sub'], size=7.5, color='#D4C8F8')

        # 功能项
        rows = d['rows']
        row_h = (h - 0.075) / len(rows)
        for ri, row in enumerate(rows):
            col_w = (w - 0.04) / len(row)
            row_y = y + h - 0.075 - (ri+1)*row_h + 0.012
            for ci, item in enumerate(row):
                bx = x + 0.02 + ci * col_w
                by = row_y
                bw = col_w - 0.012
                bh = row_h - 0.025
                rounded_box(ax, bx, by, bw, bh, WHITE, color, lw=1.2, radius=0.008)
                center_text(ax, bx+bw/2, by+bh/2, item, size=8.5, color=color, weight='bold')

    plt.tight_layout(pad=0.3)
    out = f"{OUT_DIR}/aiops_feature_diagram.png"
    plt.savefig(out, dpi=150, bbox_inches='tight', facecolor=BG)
    plt.close()
    print(f"[功能图] 已保存: {out}")
    return out


# ════════════════════════════════════════════════════════
#  图2：组件图
# ════════════════════════════════════════════════════════
def draw_component_diagram():
    fig, ax = plt.subplots(figsize=(16, 11))
    ax.set_xlim(0, 1); ax.set_ylim(0, 1)
    ax.axis('off')
    fig.patch.set_facecolor(BG)

    # ── 标题 ──────────────────────────────────────────
    rounded_box(ax, 0.02, 0.93, 0.96, 0.065, PRIMARY, PRIMARY, lw=0, radius=0.012)
    center_text(ax, 0.5, 0.963, 'AIOPS 智能运维平台 — 组件架构图',
                size=16, color=WHITE, weight='bold')

    # ════ 五层架构 ════
    layers = [
        {
            'label': '展示层',
            'y': 0.78, 'h': 0.13,
            'color': '#513CC8',
            'components': [
                ('React 18 SPA', 0.08, 0.16),
                ('Vite 构建工具', 0.27, 0.16),
                ('Tailwind CSS', 0.46, 0.16),
                ('Lucide Icons', 0.65, 0.16),
                ('Nginx 反向代理', 0.84, 0.16),
            ]
        },
        {
            'label': '应用服务层',
            'y': 0.58, 'h': 0.17,
            'color': '#3D5BD4',
            'components': [
                ('Go 1.21 + Gin\nREST API 服务', 0.08, 0.18),
                ('JWT 认证\n中间件', 0.24, 0.18),
                ('AI Agent\n推理引擎', 0.40, 0.18),
                ('WebSocket\n实时通信', 0.56, 0.18),
                ('RabbitMQ\n异步任务消费', 0.72, 0.18),
                ('GORM ORM\n数据访问', 0.88, 0.18),
            ]
        },
        {
            'label': '能力集成层',
            'y': 0.36, 'h': 0.19,
            'color': '#2E6EC4',
            'left_label': 'AI 模型接入\n(13家厂商)',
            'left_items': ['OpenAI', 'DeepSeek', '通义千问', '智谱 GLM', 'MiniMax',
                           '硅基流动', 'Kimi', '文心一言', '豆包', '混元', '百川', 'Claude', 'Gemini'],
            'right_label': '云平台接入',
            'right_items': [
                ('EasyStack', '云主机/云硬盘/网络/LB/监控\nKeystone Token 认证'),
                ('ZStack',    '虚机/存储/网络/告警\nAccessKey 认证'),
            ]
        },
        {
            'label': '数据存储层',
            'y': 0.18, 'h': 0.15,
            'color': '#1A7AB5',
            'components': [
                ('MySQL 8.0\n持久化数据库', 0.12, 0.22),
                ('RabbitMQ 3\n消息队列', 0.35, 0.22),
                ('SQLite\n开发模式', 0.58, 0.22),
                ('Docker Volume\n数据卷持久化', 0.81, 0.22),
            ]
        },
        {
            'label': '基础设施层',
            'y': 0.03, 'h': 0.12,
            'color': '#0F5AA0',
            'components': [
                ('Docker Engine', 0.10, 0.18),
                ('Docker Compose\n一键编排', 0.28, 0.18),
                ('Nginx Alpine\n容器', 0.48, 0.18),
                ('Linux 服务器\n/ 云主机', 0.67, 0.18),
                ('防火墙 / 公网IP\n网络策略', 0.86, 0.18),
            ]
        },
    ]

    for layer in layers:
        lx, ly, lw, lh = 0.03, layer['y'], 0.94, layer['h']
        color = layer['color']

        # 外框
        rounded_box(ax, lx, ly, lw, lh, PRIMARY_LIGHT, color, lw=2, radius=0.012)
        # 左侧标签
        rounded_box(ax, lx, ly, 0.10, lh, color, color, lw=0, radius=0.012)
        ax.text(lx+0.05, ly+lh/2, layer['label'],
                ha='center', va='center', fontsize=10, color=WHITE,
                fontweight='bold', rotation=90, transform=ax.transAxes)

        if 'components' in layer:
            # 普通组件行
            items = layer['components']
            total = len(items)
            avail_w = lw - 0.12
            item_w = avail_w / total - 0.008
            for i, (name, _, item_h) in enumerate(items):
                bx = lx + 0.115 + i * (avail_w / total)
                by = ly + (lh - item_h) / 2
                rounded_box(ax, bx, by, item_w, item_h, WHITE, color, lw=1.2, radius=0.008)
                center_text(ax, bx + item_w/2, by + item_h/2, name, size=8.5, color=color, weight='bold')

        elif 'left_label' in layer:
            # 能力集成层特殊布局
            # 左侧 AI 模型区域
            ai_x, ai_y = lx+0.115, ly+0.01
            ai_w, ai_h = 0.53, lh-0.02
            rounded_box(ax, ai_x, ai_y, ai_w, ai_h, '#E8E0FA', color, lw=1.2, radius=0.010)
            # AI 区域标题
            rounded_box(ax, ai_x, ai_y+ai_h-0.055, ai_w, 0.055, color, color, lw=0, radius=0.010)
            center_text(ax, ai_x+ai_w/2, ai_y+ai_h-0.027, layer['left_label'].replace('\n',' '),
                        size=9, color=WHITE, weight='bold')
            # AI 模型条目（3列×5行）
            items = layer['left_items']
            cols = 5
            rows_ai = -(-len(items) // cols)
            col_w = (ai_w - 0.02) / cols
            row_h_ai = (ai_h - 0.07) / rows_ai
            for idx, name in enumerate(items):
                ci = idx % cols
                ri = idx // cols
                bx = ai_x + 0.01 + ci * col_w
                by = ai_y + ai_h - 0.065 - (ri+1)*row_h_ai + 0.008
                bw = col_w - 0.008
                bh = row_h_ai - 0.012
                rounded_box(ax, bx, by, bw, bh, WHITE, color, lw=1.0, radius=0.006)
                center_text(ax, bx+bw/2, by+bh/2, name, size=7.8, color=color, weight='bold')

            # 右侧云平台区域
            cp_x = ai_x + ai_w + 0.015
            cp_y, cp_h = ai_y, ai_h
            cp_w = lx + lw - cp_x - 0.015
            rounded_box(ax, cp_x, cp_y, cp_w, cp_h, '#E0EAF8', color, lw=1.2, radius=0.010)
            rounded_box(ax, cp_x, cp_y+cp_h-0.055, cp_w, 0.055, color, color, lw=0, radius=0.010)
            center_text(ax, cp_x+cp_w/2, cp_y+cp_h-0.027, layer['right_label'],
                        size=9, color=WHITE, weight='bold')

            r_items = layer['right_items']
            slot_h = (cp_h - 0.07) / len(r_items)
            for idx, (name, desc) in enumerate(r_items):
                bx = cp_x + 0.01
                by = cp_y + cp_h - 0.065 - (idx+1)*slot_h + 0.008
                bw = cp_w - 0.02
                bh = slot_h - 0.014
                rounded_box(ax, bx, by, bw, bh, WHITE, color, lw=1.0, radius=0.007)
                ax.text(bx+bw/2, by+bh*0.65, name,
                        ha='center', va='center', fontsize=8.5,
                        color=color, fontweight='bold', transform=ax.transAxes)
                ax.text(bx+bw/2, by+bh*0.3, desc,
                        ha='center', va='center', fontsize=7,
                        color=GRAY_TEXT, transform=ax.transAxes)

    # 层间箭头
    for arrow_y in [0.91, 0.75, 0.55, 0.33]:
        ax.annotate('', xy=(0.5, arrow_y-0.005), xytext=(0.5, arrow_y+0.005),
                    xycoords='axes fraction', textcoords='axes fraction',
                    arrowprops=dict(arrowstyle='->', color=PRIMARY_MID,
                                   lw=1.5, mutation_scale=14))

    plt.tight_layout(pad=0.3)
    out = f"{OUT_DIR}/aiops_component_diagram.png"
    plt.savefig(out, dpi=150, bbox_inches='tight', facecolor=BG)
    plt.close()
    print(f"[组件图] 已保存: {out}")
    return out


if __name__ == '__main__':
    f1 = draw_feature_diagram()
    f2 = draw_component_diagram()
    print("完成！")
    print(f"  功能图: {f1}")
    print(f"  组件图: {f2}")
