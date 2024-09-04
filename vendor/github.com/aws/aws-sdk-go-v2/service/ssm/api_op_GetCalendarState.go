// Code generated by smithy-go-codegen DO NOT EDIT.

package ssm

import (
	"context"
	"fmt"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Gets the state of a Amazon Web Services Systems Manager change calendar at the
// current time or a specified time. If you specify a time, GetCalendarState
// returns the state of the calendar at that specific time, and returns the next
// time that the change calendar state will transition. If you don't specify a
// time, GetCalendarState uses the current time. Change Calendar entries have two
// possible states: OPEN or CLOSED .
//
// If you specify more than one calendar in a request, the command returns the
// status of OPEN only if all calendars in the request are open. If one or more
// calendars in the request are closed, the status returned is CLOSED .
//
// For more information about Change Calendar, a capability of Amazon Web Services
// Systems Manager, see [Amazon Web Services Systems Manager Change Calendar]in the Amazon Web Services Systems Manager User Guide.
//
// [Amazon Web Services Systems Manager Change Calendar]: https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-change-calendar.html
func (c *Client) GetCalendarState(ctx context.Context, params *GetCalendarStateInput, optFns ...func(*Options)) (*GetCalendarStateOutput, error) {
	if params == nil {
		params = &GetCalendarStateInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "GetCalendarState", params, optFns, c.addOperationGetCalendarStateMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*GetCalendarStateOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type GetCalendarStateInput struct {

	// The names or Amazon Resource Names (ARNs) of the Systems Manager documents (SSM
	// documents) that represent the calendar entries for which you want to get the
	// state.
	//
	// This member is required.
	CalendarNames []string

	// (Optional) The specific time for which you want to get calendar state
	// information, in [ISO 8601]format. If you don't specify a value or AtTime , the current
	// time is used.
	//
	// [ISO 8601]: https://en.wikipedia.org/wiki/ISO_8601
	AtTime *string

	noSmithyDocumentSerde
}

type GetCalendarStateOutput struct {

	// The time, as an [ISO 8601] string, that you specified in your command. If you don't
	// specify a time, GetCalendarState uses the current time.
	//
	// [ISO 8601]: https://en.wikipedia.org/wiki/ISO_8601
	AtTime *string

	// The time, as an [ISO 8601] string, that the calendar state will change. If the current
	// calendar state is OPEN , NextTransitionTime indicates when the calendar state
	// changes to CLOSED , and vice-versa.
	//
	// [ISO 8601]: https://en.wikipedia.org/wiki/ISO_8601
	NextTransitionTime *string

	// The state of the calendar. An OPEN calendar indicates that actions are allowed
	// to proceed, and a CLOSED calendar indicates that actions aren't allowed to
	// proceed.
	State types.CalendarState

	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationGetCalendarStateMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsAwsjson11_serializeOpGetCalendarState{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsjson11_deserializeOpGetCalendarState{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "GetCalendarState"); err != nil {
		return fmt.Errorf("add protocol finalizers: %v", err)
	}

	if err = addlegacyEndpointContextSetter(stack, options); err != nil {
		return err
	}
	if err = addSetLoggerMiddleware(stack, options); err != nil {
		return err
	}
	if err = addClientRequestID(stack); err != nil {
		return err
	}
	if err = addComputeContentLength(stack); err != nil {
		return err
	}
	if err = addResolveEndpointMiddleware(stack, options); err != nil {
		return err
	}
	if err = addComputePayloadSHA256(stack); err != nil {
		return err
	}
	if err = addRetry(stack, options); err != nil {
		return err
	}
	if err = addRawResponseToMetadata(stack); err != nil {
		return err
	}
	if err = addRecordResponseTiming(stack); err != nil {
		return err
	}
	if err = addClientUserAgent(stack, options); err != nil {
		return err
	}
	if err = smithyhttp.AddErrorCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = addSetLegacyContextSigningOptionsMiddleware(stack); err != nil {
		return err
	}
	if err = addTimeOffsetBuild(stack, c); err != nil {
		return err
	}
	if err = addUserAgentRetryMode(stack, options); err != nil {
		return err
	}
	if err = addOpGetCalendarStateValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opGetCalendarState(options.Region), middleware.Before); err != nil {
		return err
	}
	if err = addRecursionDetection(stack); err != nil {
		return err
	}
	if err = addRequestIDRetrieverMiddleware(stack); err != nil {
		return err
	}
	if err = addResponseErrorMiddleware(stack); err != nil {
		return err
	}
	if err = addRequestResponseLogging(stack, options); err != nil {
		return err
	}
	if err = addDisableHTTPSMiddleware(stack, options); err != nil {
		return err
	}
	return nil
}

func newServiceMetadataMiddleware_opGetCalendarState(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "GetCalendarState",
	}
}
