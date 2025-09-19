import { Module } from '@nestjs/common';
import { ProductController } from './controllers/product.controller';
import { ProductService } from './services/product.service';
import { Repositories } from './repositories/repositories';
import { Redis } from './infra/redis';
import { MessageBroker } from './infra/message-broker';
import { TypeOrmModule } from '@nestjs/typeorm';
import { Product } from './domain/product';
import { ClientsModule, Transport } from '@nestjs/microservices';

@Module({
  imports: [
    TypeOrmModule.forFeature([Product]),
    ClientsModule.register([
      {
        name: 'ORDER_EVENTS',
        transport: Transport.RMQ,
        options: {
          urls: [process.env.RABBITMQ_URL],
          queue: 'order_events_queue',
          queueOptions: { durable: true },
          exchange: 'order.exchange',
          exchangeType: 'topic',
          routingKey: 'order.*',
          wildcards: true,
        },
      },
      {
        name: 'PRODUCT_PUBLISHER',
        transport: Transport.RMQ,
        options: {
          urls: [process.env.RABBITMQ_URL],
          queue: 'product_queue',
          queueOptions: { durable: false },
        },
      },
    ]),
  ],
  controllers: [ProductController],
  providers: [ProductService, Repositories, Redis, MessageBroker],
})
export class ProductModule {}
