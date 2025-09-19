import * as dotenv from 'dotenv';
dotenv.config();
import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';
import { ValidationPipe } from '@nestjs/common';
import { MicroserviceOptions, Transport } from '@nestjs/microservices';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);

  app.useGlobalPipes(
    new ValidationPipe({
      whitelist: true,
      forbidNonWhitelisted: true,
      transform: true,
    }),
  );

  app.connectMicroservice<MicroserviceOptions>({
    transport: Transport.RMQ,
    options: {
      urls: [process.env.RABBITMQ_URL],
      queue: 'order_events_queue',
      queueOptions: {
        durable: true,
      },
      exchange: 'order.exchange',
      exchangeType: 'topic',
      routingKey: 'order.*',
      wildcards: true,
    },
  });

  await app.startAllMicroservices();
  await app.listen(3000);
}
bootstrap();
