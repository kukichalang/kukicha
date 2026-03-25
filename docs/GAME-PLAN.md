This is the complete 10-level specification for **Pipeline Panic**, a game designed to turn the Kukicha language manual into a set of physical arcade challenges.

---

# Specification: Pipeline Panic (Complete Levels 1–10)

**The Premise:** You are a "Data Plumber" in a high-concurrency production environment. Your job is to route raw data through the standard library to produce "Validated Output" while preventing "System Panics."

## I. Global Game Systems

### 1. The Operator (The Paddle)
* **Default State:** Acts as the `|>` operator. Bouncing the ball "pipes" it back into the field.
* **The `onerr` Shield:** Hold **Spacebar** to activate. This wraps the paddle in a green field (representing `onerr return`).
    * **Standard Hit:** Bounces "Healthy" data.
    * **Shield Hit:** Converts an "Error" ball back into "Healthy" data.
    * **Unshielded Error Hit:** Instant `onerr panic` (Game Over/Life Lost).

### 2. The Payload (The Ball)
A Kukicha `type Payload` that evolves:
* **Shape:** Changes based on `type` (Level 6+).
* **Color:** Indicates data state (White = String, Yellow = Int, Red = Error).
* **Physics:** Starts simple; gains complexity as it becomes a "List" (Level 4/9).

---

## II. The Level Roadmap

| Level | Title | Kukicha Concept | Gameplay Mechanic |
| :--- | :--- | :--- | :--- |
| **1** | **The Hello World** | `|>` Pipe | **Basics:** One pipe labeled `string.ToUpper`. Ball enters small/white, exits large/blue. |
| **2** | **The Bug Hunter** | `onerr panic` | **Risk:** Introduced "Flaky Pipes." They randomly eject a Red Error ball. Player must learn to use the Spacebar Shield. |
| **3** | **Function Chaining** | Pipeline Flow | **The Chain:** Pipes are stacked. Ball must hit `fetch` → `JSON` → `Clean` in sequence. Dropping it mid-chain resets the progress. |
| **4** | **Multi-Threaded** | `concurrent.Map` | **Fan-out:** A wide pipe that splits the ball into 3. The player must keep at least one ball alive to clear the level. |
| **5** | **The Firewall** | Security Debt | **Avoidance:** Red "Vulnerable" bricks (like `shell.Run`) appear. Player must avoid them or route the ball through a `shell.Safe` pipe first. |
| **6** | **The Type Factory** | `type` (Structs) | **Shape-Shifting:** A `NewUser` pipe turns the ball into a Square. A `ValidateUser` pipe only accepts Squares; Circles bounce off. |
| **7** | **The Casting Lab** | `stdlib/cast` | **Conversion:** The screen has a "Type Barrier." Only `Ints` can pass. Player must hit a `cast.SmartInt` pipe to change the ball's type. |
| **8** | **Memory Slots** | `variable` | **Storage:** Two side-pockets labeled `var A` and `var B`. Hitting them "stores" the ball. Pressing `1` or `2` spawns it back on the paddle. |
| **9** | **The Centrifuge** | `for` Loops | **Iteration:** A circular pipe track. The ball must complete 5 laps (iterations) while speeding up before the exit gate opens. |
| **10** | **The Logic Gate** | `break` / `continue` | **Control Flow:** Inside the Level 9 loop, "Bumper" switches appear. Hit `continue` to skip a hazard; hit `break` to exit early once a "Goal" is met. |

---

## III. Technical Component Specification

### 1. Data Structures (Kukicha)
The game state relies on a central `type` to manage the ball's meta-data as it is transformed by the "stdlib" pipes.

```kukicha
type Payload
    body game.Circle
    typeName string        # "raw", "string", "int", "User"
    isError bool
    isList bool
    iteration int          # For Level 9 loops
    lastFunc string        # Name of the last pipe hit
```

### 2. Pipe Physics: "The Intake"
Unlike the `08-breakout` bricks that just bounce, Pipes use a "Suck and Shoot" mechanic.

```kukicha
function handlePipeIntake(ball reference Payload, p Pipe)
    # Check if ball hits the hollow center of the pipe
    if game.CircleOverlapsRect(ball.body, p.IntakeBounds)
        ball.IsSucked = true
        ball.PhysicsMode = "Path" # Ball follows pipe's internal path
        # Apply transformation logic
        applyTransform(ball, p.FunctionName)
```

### 3. The `onerr` Handler Logic
This logic governs the interaction between the paddle and the ball based on the shield state.

```kukicha
function onPaddleCollision()
    isShielded := game.IsKeyDown(game.KeySpace)
    
    if ball.isError
        if isShielded
            # Handled! (onerr return)
            ball.isError = false
            ball.Color = game.White
            score = score + 100
        else
            # Unhandled! (onerr panic)
            triggerPanic()
```

---

## IV. Visual Language (UX)
* **The Terminal Aesthetic:** The background is dark (`game.MakeColor(20, 22, 25, 255)`).
* **The Code Font:** Function names on pipes use a monospace font via `game.DrawText`.
* **The "Compile" Effect:** When a ball changes type (e.g., in Level 7), it flashes white with a "Success" sound effect.
* **The Panic Effect:** If an error isn't caught, the screen briefly turns red and displays the Kukicha error stack trace before resetting.

