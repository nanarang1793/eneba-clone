import { Link, Navigate, Route, Routes } from "react-router";
import { Layout } from "../components/layout/Layout";
import { AccountPage } from "../features/account/AccountPage";
import { LoginPage } from "../features/auth/LoginPage";
import { RegisterPage } from "../features/auth/RegisterPage";
import { RequireAuth } from "../features/auth/RequireAuth";
import { CartPage } from "../features/cart/CartPage";
import { CheckoutPage } from "../features/checkout/CheckoutPage";
import { OrderPage } from "../features/orders/OrderPage";
import { ProductPage } from "../features/product/ProductPage";
import { StorePage } from "../features/store/StorePage";

export function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        {/* The data-driven homepage comes on day 6; until then / opens the store. */}
        <Route index element={<Navigate to="/store" replace />} />
        <Route path="store" element={<StorePage />} />
        <Route path="product/:slug" element={<ProductPage />} />
        <Route path="cart" element={<CartPage />} />
        <Route
          path="checkout/:id"
          element={
            <RequireAuth>
              <CheckoutPage />
            </RequireAuth>
          }
        />
        <Route
          path="orders/:id"
          element={
            <RequireAuth>
              <OrderPage />
            </RequireAuth>
          }
        />
        <Route path="login" element={<LoginPage />} />
        <Route path="register" element={<RegisterPage />} />
        <Route
          path="account"
          element={
            <RequireAuth>
              <AccountPage />
            </RequireAuth>
          }
        />
        <Route path="*" element={<NotFound />} />
      </Route>
    </Routes>
  );
}

function NotFound() {
  return (
    <div className="py-24 text-center">
      <h1 className="text-3xl font-bold">Page not found</h1>
      <Link to="/store" className="mt-4 inline-block text-accent hover:underline">
        Go to the store
      </Link>
    </div>
  );
}
