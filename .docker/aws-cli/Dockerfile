FROM amazon/aws-cli:2.3.7

COPY .docker/aws-cli/start.sh /usr/local/bin/start.sh

RUN alias laws='aws --endpoint-url=http://localstack:4566'

#ENTRYPOINT [ "sh" ]

#CMD [ "/usr/local/bin/start.sh" ]

ENTRYPOINT [ "tail" ]

CMD [ "-f", "/dev/null" ]
