# Markdown 图表与图形能力测试文档

> 用途：测试 Markdown 渲染器对各种图表、图形、表格、代码块的渲染支持情况
> 测试日期：2025 年 7 月

---

## 目录

1. [Mermaid 流程图](#1-mermaid-流程图)
2. [Mermaid 时序图](#2-mermaid-时序图)
3. [Mermaid 类图](#3-mermaid-类图)
4. [Mermaid 状态图](#4-mermaid-状态图)
5. [Mermaid 甘特图](#5-mermaid-甘特图)
6. [Mermaid 饼图](#6-mermaid-饼图)
7. [Mermaid ER 图](#7-mermaid-er-图)
8. [Mermaid 用户旅程图](#8-mermaid-用户旅程图)
9. [Mermaid 思维导图](#9-mermaid-思维导图)
10. [Mermaid  Gitgraph](#10-mermaid-gitgraph)
11. [表格](#11-表格)
12. [代码块语法高亮](#12-代码块语法高亮)
13. [数学公式](#13-数学公式)
14. [HTML 内嵌图形](#14-html-内嵌图形)
15. [嵌套列表与任务列表](#15-嵌套列表与任务列表)

---

## 1. Mermaid 流程图

### 1.1 基本流程图

```mermaid
flowchart TD
    A[开始] --> B{是否登录?}
    B -->|是| C[访问主页]
    B -->|否| D[跳转登录页]
    D --> E[输入账号密码]
    E --> F{验证通过?}
    F -->|是| C
    F -->|否| G[显示错误提示]
    G --> E
    C --> H[结束]
```

### 1.2 子流程与样式

```mermaid
flowchart LR
    subgraph 前端
        A[Vue 组件] --> B[API 调用]
    end
    subgraph 后端
        C[路由分发] --> D[业务逻辑]
        D --> E[(数据库)]
    end
    subgraph 第三方
        F[支付网关]
        G[短信服务]
    end
    B --> C
    D --> F
    D --> G
    
    style A fill:#f9f,stroke:#333,stroke-width:2px
    style D fill:#bbf,stroke:#f66,stroke-width:2px
    style E fill:#ff9,stroke:#333,stroke-width:2px
```

### 1.3 复杂分支

```mermaid
flowchart TD
    Start([开始]) --> Input[输入 n]
    Input --> Check{n <= 0?}
    Check -->|是| Error[提示输入无效]
    Error --> End([结束])
    Check -->|否| Init[i = 1, sum = 0]
    Init --> Loop{i <= n?}
    Loop -->|是| Add[sum = sum + i]
    Add --> Inc[i = i + 1]
    Inc --> Loop
    Loop -->|否| Output[输出 sum]
    Output --> End
```

---

## 2. Mermaid 时序图

### 2.1 用户登录时序

```mermaid
sequenceDiagram
    actor 用户
    participant 浏览器
    participant 后端API
    participant 数据库
    
    用户->>浏览器: 输入账号密码
    浏览器->>后端API: POST /api/login
    后端API->>数据库: 查询用户
    数据库-->>后端API: 返回用户信息
    后端API->>后端API: 验证密码
    后端API->>后端API: 生成 JWT Token
    后端API-->>浏览器: 200 OK + Token
    浏览器->>浏览器: 存储 Token
    浏览器-->>用户: 跳转首页
```

### 2.2 订单流程

```mermaid
sequenceDiagram
    participant 客户
    participant 商城
    participant 库存
    participant 支付
    
    客户->>商城: 提交订单
    商城->>库存: 锁定库存
    库存-->>商城: 锁定成功
    商城->>支付: 发起支付请求
    支付->>支付: 处理支付
    支付-->>商城: 支付成功回调
    商城->>库存: 扣减库存
    商城-->>客户: 订单确认
    客户->>商城: 查询订单状态
    商城-->>客户: 已支付/待发货
```

---

## 3. Mermaid 类图

### 3.1 电商系统模型

```mermaid
classDiagram
    class User {
        +String id
        +String name
        +String email
        +login() bool
        +logout() void
        +getOrders() List~Order~
    }
    
    class Order {
        +String id
        +Date createTime
        +float totalAmount
        +String status
        +pay() bool
        +cancel() void
    }
    
    class Product {
        +String id
        +String name
        +float price
        +int stock
        +reduceStock(int) bool
    }
    
    class OrderItem {
        +String id
        +int quantity
        +float subtotal
    }
    
    class Payment {
        +String id
        +float amount
        +String method
        +Date payTime
        +String transactionId
        +refund() bool
    }
    
    User "1" --> "*" Order : 拥有
    Order "1" --> "*" OrderItem : 包含
    Order "1" --> "1" Payment : 关联
    OrderItem "*" --> "1" Product : 引用
```

### 3.2 继承与接口

```mermaid
classDiagram
    Animal <|-- Dog
    Animal <|-- Cat
    Animal <|-- Bird
    Animal : +String name
    Animal : +int age
    Animal : +eat() void
    Animal : +sleep() void
    
    Flyable <|.. Bird
    Flyable <|.. Insect
    Flyable : +fly() void
    
    class Dog {
        +bark() void
        +fetchBall() void
    }
    
    class Cat {
        +meow() void
        +climbTree() void
    }
    
    class Bird {
        +sing() void
        +fly() void
    }
    
    class Insect {
        +fly() void
        +metamorphosis() void
    }
```

---

## 4. Mermaid 状态图

### 4.1 订单状态流转

```mermaid
stateDiagram-v2
    [*] --> 待支付
    待支付 --> 已取消: 用户取消
    待支付 --> 已支付: 支付成功
    已支付 --> 备货中
    备货中 --> 已发货: 仓库出库
    已发货 --> 运输中
    运输中 --> 已签收: 客户签收
    已签收 --> 已完成
    已完成 --> [*]
    
    已支付 --> 退款中: 申请退款
    退款中 --> 已退款: 审核通过
    已退款 --> [*]
    退款中 --> 已支付: 取消退款
```

### 4.2 程序生命周期

```mermaid
stateDiagram-v2
    state "已停止" as stopped
    state "运行中" as running
    state "暂停" as paused
    state "错误" as error_state
    
    [*] --> stopped: 安装
    stopped --> running: start()
    running --> stopped: stop()
    running --> paused: pause()
    paused --> running: resume()
    running --> error_state: 异常
    error_state --> running: recover()
    error_state --> stopped: 强制终止
```

---

## 5. Mermaid 甘特图

### 5.1 项目开发计划

```mermaid
gantt
    title 软件开发周期
    dateFormat  YYYY-MM-DD
    axisFormat  %m-%d
    
    section 需求阶段
    需求调研           :a1, 2025-01-01, 14d
    需求评审           :a2, after a1, 3d
    
    section 设计阶段
    架构设计           :b1, after a2, 7d
    数据库设计          :b2, after a2, 5d
    UI/UX 设计         :b3, after a2, 10d
    
    section 开发阶段
    前端开发           :c1, after b3, 20d
    后端开发           :c2, after b1, 25d
    接口联调           :c3, after c1, 5d
    
    section 测试阶段
    单元测试           :d1, after c2, 5d
    集成测试           :d2, after c3, 7d
    压力测试           :d3, after d2, 3d
    
    section 发布阶段
    预发布             :e1, after d2, 2d
    正式上线           :e2, after e1, 1d
```

---

## 6. Mermaid 饼图

### 6.1 技术栈分布

```mermaid
pie
    title 项目技术栈占比
    "Go" : 35
    "Vue.js" : 25
    "Python" : 15
    "TypeScript" : 10
    "Docker/K8s" : 10
    "其他" : 5
```

### 6.2 时间分配

```mermaid
pie showData
    title 工作日时间分配
    "编码" : 40
    "会议" : 15
    "代码审查" : 10
    "文档编写" : 10
    "学习研究" : 15
    "其他事务" : 10
```

---

## 7. Mermaid ER 图

### 7.1 数据库模型

```mermaid
erDiagram
    User ||--o{ Order : "下单"
    User {
        string id PK
        string name
        string email
        string phone
        datetime created_at
    }
    Order ||--|{ OrderItem : "包含"
    Order {
        string id PK
        string user_id FK
        float total_amount
        string status
        datetime created_at
    }
    Product ||--o{ OrderItem : "属于"
    Product {
        string id PK
        string name
        string category
        float price
        int stock
    }
    OrderItem {
        string id PK
        string order_id FK
        string product_id FK
        int quantity
        float subtotal
    }
    Category ||--o{ Product : "分类"
    Category {
        string id PK
        string name
        string parent_id FK
    }
```

---

## 8. Mermaid 用户旅程图

```mermaid
journey
    title 用户购物体验
    section 浏览商品
        打开应用: 5: 用户
        搜索商品: 4: 用户
        浏览推荐: 3: 用户, 系统
    section 下单支付
        加入购物车: 4: 用户
        提交订单: 3: 用户, 系统
        完成支付: 2: 用户, 支付系统
    section 收货售后
        查看物流: 3: 用户
        确认收货: 4: 用户
        评价商品: 3: 用户
```

---

## 9. Mermaid 思维导图

```mermaid
mindmap
  root((项目架构))
    前端层
      Vue 3
        Composition API
        Pinia 状态管理
      Element Plus
      构建工具
        Vite
        ESLint
    后端层
      Go 语言
        Gin 框架
        GORM ORM
      中间件
        Redis 缓存
        RabbitMQ 消息队列
      认证
        JWT
        OAuth2.0
    数据层
      PostgreSQL
      MongoDB
    DevOps
      Docker
      K8s
      CI/CD
```

---

## 10. Mermaid Gitgraph

```mermaid
gitGraph
    commit id: "初始化项目"
    commit id: "搭建基础框架"
    branch dev
    checkout dev
    commit id: "开发登录功能"
    commit id: "开发用户管理"
    branch feature/payment
    checkout feature/payment
    commit id: "集成支付网关"
    commit id: "支付回调处理"
    checkout dev
    merge feature/payment
    branch release/v1.0
    checkout release/v1.0
    commit id: "修复支付BUG" type: HIGHLIGHT
    checkout main
    merge release/v1.0 tag: "v1.0.0"
    checkout dev
    commit id: "继续开发v2.0"
```

---

## 11. 表格

### 11.1 对齐测试

| 左对齐 | 居中 | 右对齐 | 默认 |
|:-------|:----:|-------:|------|
| 苹果   | 红色 |   ￥5  | 水果  |
| 香蕉   | 黄色 |   ￥3  | 水果  |
| 西瓜   | 绿色 |  ￥15  | 水果  |
| 牛肉   | 红色 |   ￥45 | 肉类  |

### 11.2 带格式表格

| 功能 | 优先级 | 状态 | 负责人 | 预计工时 | 截止日期 |
|:-----|:------:|:----:|:------:|:--------:|:--------:|
| 用户登录 | **P0** | ✅ 已完成 | 张三 | 3d | 2025-01-10 |
| 支付接口 | **P0** | 🔄 开发中 | 李四 | 5d | 2025-01-15 |
| 数据导出 | P2 | ⏳ 待开始 | 王五 | 2d | 2025-01-20 |
| 性能优化 | P1 | ❌ 已阻塞 | 赵六 | 4d | 2025-01-18 |
| ~~旧版迁移~~ | P3 | ~~已取消~~ | - | - | - |

### 11.3 长文本表格

| 模块 | 描述 | 关键技术 | 备注 |
|:-----|:-----|:---------|:-----|
| 认证中心 | 负责用户身份认证、Token 签发与验证、OAuth2.0 第三方登录集成 | JWT, Redis, OAuth2 | 需对接微信/支付宝 |
| 消息队列 | 异步处理订单通知、短信发送、日志采集等非实时任务，削峰填谷 | RabbitMQ, 死信队列 | 消息持久化到磁盘 |
| 网关路由 | 统一入口、限流熔断、请求转发、日志记录、跨域处理 | Gin 中间件, 令牌桶 | 参考 Kubernetes Ingress |

---

## 12. 代码块语法高亮

### 12.1 Go

```go
package main

import (
    "context"
    "fmt"
    "time"
)

// User 结构体
type User struct {
    ID        int64     `json:"id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
}

// Greet 方法
func (u *User) Greet() string {
    return fmt.Sprintf("Hello, I'm %s (ID: %d)", u.Name, u.ID)
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    user := &User{ID: 1, Name: "Alice", CreatedAt: time.Now()}
    fmt.Println(user.Greet())

    select {
    case <-ctx.Done():
        fmt.Println("timeout")
    default:
        fmt.Println("done")
    }
}
```

### 12.2 TypeScript

```typescript
import { Component, Vue } from 'vue-property-decorator'
import { UserService } from '@/services/user'

interface ApiResponse<T> {
  code: number
  message: string
  data: T | null
}

@Component
export default class UserList extends Vue {
  private users: Array<{ id: number; name: string }> = []
  private loading = false

  async fetchUsers(): Promise<void> {
    this.loading = true
    try {
      const res = await UserService.list<ApiResponse<any>>()
      if (res.code === 200) {
        this.users = res.data ?? []
      }
    } catch (err) {
      console.error('加载用户失败:', err)
    } finally {
      this.loading = false
    }
  }
}
```

### 12.3 Python

```python
from datetime import datetime
from typing import Optional
import asyncio


class DataProcessor:
    """数据处理基类"""
    
    def __init__(self, name: str, batch_size: int = 100):
        self.name = name
        self.batch_size = batch_size
        self.processed_count = 0
    
    async def process_batch(self, items: list) -> list[dict]:
        """异步批量处理数据"""
        results = []
        for item in items:
            processed = await self._process_single(item)
            results.append(processed)
            self.processed_count += 1
        return results
    
    async def _process_single(self, item) -> dict:
        """单个数据处理（子类实现）"""
        raise NotImplementedError


class JsonProcessor(DataProcessor):
    """JSON 数据处理器"""
    
    async def _process_single(self, item) -> dict:
        return {
            "raw": item,
            "timestamp": datetime.utcnow().isoformat(),
            "length": len(str(item)),
        }


async def main():
    processor = JsonProcessor("json-proc", batch_size=50)
    data = [{"id": i, "value": f"item-{i}"} for i in range(10)]
    results = await processor.process_batch(data)
    print(f"已处理 {processor.processed_count} 条数据")


if __name__ == "__main__":
    asyncio.run(main())
```

### 12.4 SQL

```sql
-- 用户订单统计查询
WITH user_orders AS (
    SELECT
        u.id AS user_id,
        u.name AS user_name,
        COUNT(o.id) AS order_count,
        SUM(o.total_amount) AS total_spent
    FROM users u
    LEFT JOIN orders o ON u.id = o.user_id
    WHERE o.created_at >= '2025-01-01'
    GROUP BY u.id, u.name
    HAVING COUNT(o.id) > 0
),
ranked_users AS (
    SELECT
        *,
        ROW_NUMBER() OVER (ORDER BY total_spent DESC) AS rank
    FROM user_orders
)
SELECT
    rank,
    user_name,
    order_count,
    ROUND(total_spent, 2) AS total_spent
FROM ranked_users
WHERE rank <= 10
ORDER BY rank;
```

### 12.5 JSON

```json
{
  "swagger": "2.0",
  "info": {
    "title": "电商系统 API",
    "version": "1.0.0"
  },
  "paths": {
    "/api/v1/users": {
      "get": {
        "summary": "获取用户列表",
        "parameters": [
          {
            "name": "page",
            "in": "query",
            "type": "integer",
            "default": 1
          },
          {
            "name": "page_size",
            "in": "query",
            "type": "integer",
            "default": 20
          }
        ],
        "responses": {
          "200": {
            "description": "成功返回用户列表",
            "schema": {
              "type": "object",
              "properties": {
                "code": { "type": "integer" },
                "data": {
                  "type": "array",
                  "items": { "$ref": "#/definitions/User" }
                }
              }
            }
          }
        }
      }
    }
  },
  "definitions": {
    "User": {
      "type": "object",
      "properties": {
        "id": { "type": "integer" },
        "name": { "type": "string" },
        "email": { "type": "string" }
      }
    }
  }
}
```

### 12.6 行内代码

在 `Go` 中，使用 `goroutine` 实现并发：`go func() { ... }()`。
通过 `channel` 通信：`ch := make(chan struct{})`。
错误处理使用 `if err != nil { return err }` 模式。

---

## 13. 数学公式

### 13.1 内联公式

勾股定理：$a^2 + b^2 = c^2$
欧拉公式：$e^{i\pi} + 1 = 0$
正态分布：$X \sim \mathcal{N}(\mu, \sigma^2)$

### 13.2 块级公式

$$
\frac{-b \pm \sqrt{b^2 - 4ac}}{2a}
$$

$$
\int_{-\infty}^{\infty} e^{-x^2} \, dx = \sqrt{\pi}
$$

$$
\sum_{n=1}^{\infty} \frac{1}{n^2} = \frac{\pi^2}{6}
$$

$$
\mathbf{H} = \begin{pmatrix}
\frac{\partial^2 f}{\partial x_1^2} & \frac{\partial^2 f}{\partial x_1 \partial x_2} \\
\frac{\partial^2 f}{\partial x_2 \partial x_1} & \frac{\partial^2 f}{\partial x_2^2}
\end{pmatrix}
$$

---

## 14. HTML 内嵌图形

> 以下用 HTML 模拟进度条、色块、布局等图形效果（部分 Markdown 渲染器支持）

<div style="display: flex; gap: 20px; flex-wrap: wrap;">

<div style="flex: 1; min-width: 200px; padding: 15px; border: 1px solid #ddd; border-radius: 8px;">

### 进度条

<div style="background: #eee; border-radius: 10px; padding: 3px; margin: 10px 0;">
  <div style="width: 80%; height: 20px; background: linear-gradient(90deg, #4CAF50, #8BC34A); border-radius: 8px; text-align: center; line-height: 20px; color: #fff; font-size: 12px;">80%</div>
</div>

<div style="background: #eee; border-radius: 10px; padding: 3px; margin: 10px 0;">
  <div style="width: 45%; height: 20px; background: linear-gradient(90deg, #FF9800, #FFC107); border-radius: 8px; text-align: center; line-height: 20px; color: #fff; font-size: 12px;">45%</div>
</div>

<div style="background: #eee; border-radius: 10px; padding: 3px; margin: 10px 0;">
  <div style="width: 15%; height: 20px; background: linear-gradient(90deg, #F44336, #E91E63); border-radius: 8px; text-align: center; line-height: 20px; color: #fff; font-size: 12px;">15%</div>
</div>

</div>

<div style="flex: 1; min-width: 200px; padding: 15px; border: 1px solid #ddd; border-radius: 8px;">

### 色卡

<div style="display: flex; gap: 5px; flex-wrap: wrap;">
  <div style="width: 40px; height: 40px; background: #F44336; border-radius: 4px;" title="红色 #F44336"></div>
  <div style="width: 40px; height: 40px; background: #FF9800; border-radius: 4px;" title="橙色 #FF9800"></div>
  <div style="width: 40px; height: 40px; background: #FFEB3B; border-radius: 4px;" title="黄色 #FFEB3B"></div>
  <div style="width: 40px; height: 40px; background: #4CAF50; border-radius: 4px;" title="绿色 #4CAF50"></div>
  <div style="width: 40px; height: 40px; background: #2196F3; border-radius: 4px;" title="蓝色 #2196F3"></div>
  <div style="width: 40px; height: 40px; background: #9C27B0; border-radius: 4px;" title="紫色 #9C27B0"></div>
  <div style="width: 40px; height: 40px; background: #795548; border-radius: 4px;" title="棕色 #795548"></div>
  <div style="width: 40px; height: 40px; background: #607D8B; border-radius: 4px;" title="蓝灰 #607D8B"></div>
</div>

</div>

</div>

---

## 15. 嵌套列表与任务列表

### 15.1 多级嵌套

- 编程语言
  - 编译型
    - Go
    - Rust
    - C/C++
  - 解释型
    - Python
    - JavaScript
      - ES6+
      - TypeScript
    - Ruby
- 数据库
  - 关系型
    1. PostgreSQL
    2. MySQL
    3. SQLite
  - NoSQL
    1. MongoDB
    2. Redis
    3. Elasticsearch

### 15.2 任务列表

- [x] 完成架构设计
- [x] 实现用户登录模块
- [ ] 集成支付接口
  - [x] 支付宝对接
  - [ ] 微信支付对接
- [ ] 编写 API 文档
- [ ] 部署测试环境
  - [x] 申请服务器
  - [ ] 配置域名
  - [ ] 部署 Docker 容器

### 15.3 混合列表

1. 第一阶段：基础设施
   - [x] 搭建开发环境
   - [x] 配置 CI/CD 流水线
   - [ ] ~~购买独立服务器~~（改用云服务）
2. 第二阶段：核心功能
   - [x] 用户认证
   - [ ] 商品管理
     - 商品 CRUD 接口
     - 商品搜索（需要 Elasticsearch）
3. 第三阶段：上线准备
   - [ ] 压力测试
   - [ ] 安全审计

---

## 引用与分隔线

### 引用嵌套

> **《道德经》**
>
> 道可道，非常道；名可名，非常名。
>
> > 无名天地之始，有名万物之母。
> >
> > > 故常无欲，以观其妙；常有欲，以观其徼。

### 分隔线变体

普通分隔线：

---

带星号分隔线：

***

带下划线分隔线：

___

---

## 脚注与链接

这是一段需要脚注的文字[^1]，这里还有另一个脚注[^2]。

[^1]: 这是第一个脚注的内容。
[^2]: 这是第二个脚注的内容，可以包含**格式**和`代码`。

---

## 图片与媒体

### 占位图测试

![占位图 - 320x200](https://via.placeholder.com/320x200/4CAF50/FFFFFF?text=Markdown+Test)

### SVG 内嵌

> 以下是用 Markdown 直接写的 SVG 图形（部分渲染器支持）

```svg
<svg width="300" height="100" xmlns="http://www.w3.org/2000/svg">
  <rect x="10" y="10" width="280" height="80" rx="10" fill="#f0f0f0" stroke="#333" stroke-width="2"/>
  <text x="150" y="50" font-family="Arial" font-size="20" text-anchor="middle" fill="#333">
    内嵌 SVG 文本
  </text>
  <circle cx="50" cy="75" r="8" fill="#4CAF50"/>
  <circle cx="150" cy="75" r="8" fill="#FF9800"/>
  <circle cx="250" cy="75" r="8" fill="#F44336"/>
</svg>
```

---

## 总结

| 测试项 | 状态 | 备注 |
|:-------|:----:|:-----|
| Mermaid 流程图 | 🔲 | 需渲染器支持 mermaid 块 |
| Mermaid 时序图 | 🔲 | 同上 |
| Mermaid 类图 | 🔲 | 同上 |
| Mermaid 状态图 | 🔲 | 同上 |
| Mermaid 甘特图 | 🔲 | 同上 |
| Mermaid 饼图 | 🔲 | 同上 |
| Mermaid ER 图 | 🔲 | 同上 |
| 表格对齐 | 🔲 | 标准 Markdown |
| 代码高亮 | 🔲 | 需语法高亮引擎 |
| 数学公式 | 🔲 | 需 KaTeX/MathJax |
| HTML 内嵌 | 🔲 | 部分渲染器支持 |
| 嵌套列表 | 🔲 | 标准 Markdown |
| 任务列表 | 🔲 | GFM 扩展 |
| SVG 内嵌 | 🔲 | 部分渲染器支持 |

> ⚠️ 说明：请根据你的 Markdown 渲染器实际能力，在上述表格中填入 ✅ 或 ❌。

---

*测试文档结束 · 共包含 10 类 Mermaid 图形 + 表格 + 代码高亮 + 公式 + HTML 图形 + 列表嵌套*
