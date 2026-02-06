// ======================================================
// FILE: backend/go.mod
// ======================================================
module ecommerce

go 1.22

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/jinzhu/gorm v1.9.16
	github.com/jinzhu/gorm/dialects/sqlite v1.9.16
	github.com/google/uuid v1.6.0
)

// ======================================================
// FILE: backend/main.go
// ======================================================
package main

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/google/uuid"
)

var db *gorm.DB

type User struct {
	gorm.Model
	Username string `gorm:"unique"`
	Password string
	Token    string
}

type Item struct {
	gorm.Model
	Name  string
	Price float64
}

type Cart struct {
	gorm.Model
	UserID uint
	Items  []CartItem
}

type CartItem struct {
	gorm.Model
	CartID uint
	ItemID uint
}

type Order struct {
	gorm.Model
	UserID uint
	Items  []OrderItem
}

type OrderItem struct {
	gorm.Model
	OrderID uint
	ItemID  uint
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		var user User
		if db.Where("token = ?", token).First(&user).RecordNotFound() {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}
		c.Set("user", user)
		c.Next()
	}
}

func main() {
	var err error
	db, err = gorm.Open("sqlite3", "shop.db")
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&User{}, &Item{}, &Cart{}, &CartItem{}, &Order{}, &OrderItem{})

	r := gin.Default()

	r.POST("/users", createUser)
	r.GET("/users", listUsers)
	r.POST("/users/login", loginUser)

	r.POST("/items", createItem)
	r.GET("/items", listItems)

	r.POST("/carts", AuthMiddleware(), addToCart)
	r.GET("/carts", AuthMiddleware(), listCarts)

	r.POST("/orders", AuthMiddleware(), createOrder)
	r.GET("/orders", AuthMiddleware(), listOrders)

	r.Run(":8080")
}

func createUser(c *gin.Context) {
	var u User
	c.BindJSON(&u)
	db.Create(&u)
	c.JSON(200, u)
}

func listUsers(c *gin.Context) {
	var users []User
	db.Find(&users)
	c.JSON(200, users)
}

func loginUser(c *gin.Context) {
	var input User
	c.BindJSON(&input)

	var user User
	if db.Where("username=? AND password=?", input.Username, input.Password).First(&user).RecordNotFound() {
		c.JSON(401, gin.H{"error": "Invalid username/password"})
		return
	}

	token := uuid.New().String()
	user.Token = token
	db.Save(&user)

	c.JSON(200, gin.H{"token": token})
}

func createItem(c *gin.Context) {
	var item Item
	c.BindJSON(&item)
	db.Create(&item)
	c.JSON(200, item)
}

func listItems(c *gin.Context) {
	var items []Item
	db.Find(&items)
	c.JSON(200, items)
}

func addToCart(c *gin.Context) {
	user := c.MustGet("user").(User)

	var body struct {
		ItemID uint
	}
	c.BindJSON(&body)

	var cart Cart
	if db.Where("user_id=?", user.ID).First(&cart).RecordNotFound() {
		cart = Cart{UserID: user.ID}
		db.Create(&cart)
	}

	ci := CartItem{CartID: cart.ID, ItemID: body.ItemID}
	db.Create(&ci)

	c.JSON(200, ci)
}

func listCarts(c *gin.Context) {
	user := c.MustGet("user").(User)
	var carts []Cart
	db.Preload("Items").Where("user_id=?", user.ID).Find(&carts)
	c.JSON(200, carts)
}

func createOrder(c *gin.Context) {
	user := c.MustGet("user").(User)

	var body struct{ CartID uint }
	c.BindJSON(&body)

	var cart Cart
	db.Preload("Items").First(&cart, body.CartID)

	order := Order{UserID: user.ID}
	db.Create(&order)

	for _, ci := range cart.Items {
		oi := OrderItem{OrderID: order.ID, ItemID: ci.ItemID}
		db.Create(&oi)
	}

	db.Where("cart_id=?", cart.ID).Delete(CartItem{})

	c.JSON(200, order)
}

func listOrders(c *gin.Context) {
	user := c.MustGet("user").(User)
	var orders []Order
	db.Preload("Items").Where("user_id=?", user.ID).Find(&orders)
	c.JSON(200, orders)
}

// ======================================================
// FILE: frontend/package.json
// ======================================================
{
  "name": "frontend",
  "version": "1.0.0",
  "private": true,
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-scripts": "5.0.1"
  },
  "scripts": {
    "start": "react-scripts start",
    "build": "react-scripts build"
  }
}

// ======================================================
// FILE: frontend/src/App.js
// ======================================================
import { useState, useEffect } from "react";

const API = "http://localhost:8080";

export default function App() {
  const [token, setToken] = useState("");
  const [items, setItems] = useState([]);

  const login = async (u, p) => {
    const r = await fetch(API+"/users/login", {
      method: "POST",
      headers: {"Content-Type":"application/json"},
      body: JSON.stringify({username:u,password:p})
    });
    if (!r.ok) return alert("Invalid username/password");
    const d = await r.json();
    setToken(d.token);
  };

  const loadItems = async () => {
    const r = await fetch(API+"/items");
    setItems(await r.json());
  };

  useEffect(()=>{ if(token) loadItems(); },[token]);

  if (!token) return <Login onLogin={login}/>;
  return <Items items={items} token={token}/>;
}

function Login({onLogin}) {
  const [u,setU]=useState("");
  const [p,setP]=useState("");
  return (
    <div>
      <h2>Login</h2>
      <input placeholder="username" onChange={e=>setU(e.target.value)}/>
      <input placeholder="password" type="password" onChange={e=>setP(e.target.value)}/>
      <button onClick={()=>onLogin(u,p)}>Login</button>
    </div>
  );
}

function Items({items, token}) {
  const add = async (id) => {
    await fetch(API+"/carts", {
      method:"POST",
      headers:{"Content-Type":"application/json", Authorization:token},
      body: JSON.stringify({itemID:id})
    });
    alert("Added to cart");
  };

  const showCart = async () => {
    const r = await fetch(API+"/carts", {headers:{Authorization:token}});
    const d = await r.json();
    alert(JSON.stringify(d));
  };

  const showOrders = async () => {
    const r = await fetch(API+"/orders", {headers:{Authorization:token}});
    const d = await r.json();
    alert(JSON.stringify(d.map(o=>o.ID)));
  };

  const checkout = async () => {
    const r = await fetch(API+"/carts", {headers:{Authorization:token}});
    const carts = await r.json();
    if (!carts.length) return alert("No cart");

    await fetch(API+"/orders", {
      method:"POST",
      headers:{"Content-Type":"application/json", Authorization:token},
      body: JSON.stringify({cartID:carts[0].ID})
    });
    alert("Order successful");
  };

  return (
    <div>
      <button onClick={checkout}>Checkout</button>
      <button onClick={showCart}>Cart</button>
      <button onClick={showOrders}>Order History</button>
      <h3>Items</h3>
      {items.map(i=> (
        <div key={i.ID} onClick={()=>add(i.ID)}>
          {i.Name} - {i.Price}
        </div>
      ))}
    </div>
  );
}
 