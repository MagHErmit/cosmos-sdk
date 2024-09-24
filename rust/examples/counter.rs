#![allow(missing_docs)]
#[ixc::account_handler(Counter)]
pub mod counter {
    use ixc::*;

    #[derive(Resources)]
    pub struct Counter {
        value: Item<u64>,
    }

    impl Counter {
        #[on_create]
        pub fn create(ctx: &mut Context) {
        }

        #[publish]
        pub fn get(&self, ctx: &Context) -> Response<u64> {
            // self.value.get(ctx)
            todo!()
        }

        #[publish]
        pub fn inc(&mut self, ctx: &mut Context) -> Response<()> {
            // let value = self.value.get(ctx)?;
            // let new_value = value.checked_add(1).ok_or(())?;
            // self.value.set(ctx, new_value)
            todo!()
        }
    }
}

#[cfg(test)]
mod tests {
    use ixc_testing::*;
    use super::counter::*;

    #[test]
    fn test_counter() {
        let mut app = TestApp::default();
        let alice = app.new_client_address();
        let counter_inst = app.add_account::<Counter>(&alice, ()).unwrap();
    }
}

fn main() {}